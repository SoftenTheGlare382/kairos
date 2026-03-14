package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"kairos/pkg/config"
	"kairos/pkg/grpc"
	"kairos/pkg/middleware"
	"kairos/pkg/rabbitmq"
	"kairos/pkg/redis"
	im "kairos/services/im/internal"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	config.LoadEnvFromSearchPaths(true)
	cfg := config.Load()

	db, err := openDB(cfg.Database)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := db.AutoMigrate(&im.Conversation{}, &im.Message{}, &im.ConversationRead{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	if cfg.RabbitMQ.URL == "" {
		log.Fatalf("RABBITMQ_URL is required for IM async persistence")
	}
	mq, err := rabbitmq.New(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	defer mq.Close()

	// 预先声明 IM 消息队列（带 DLQ），避免 Publish 用简单声明导致 PRECONDITION_FAILED
	if err := mq.EnsureQueueWithDLQ(im.QueueIMMessagePersist, im.QueueIMMessagePersistDLQ); err != nil {
		log.Fatalf("rabbitmq ensure im queue: %v", err)
	}

	rdb := redis.New(redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()); err != nil {
		log.Fatalf("redis required: %v", err)
	}
	defer rdb.Close()

	accountAddr := cfg.IM.AccountGrpcAddr
	if accountAddr == "" {
		accountAddr = cfg.Account.GrpcAddr
	}
	accountCli, err := grpc.NewAccountClient(accountAddr)
	if err != nil {
		log.Fatalf("account gRPC: %v", err)
	}
	defer accountCli.Close()

	var socialCli im.SocialClient
	socialAddr := cfg.IM.SocialGrpcAddr
	if socialAddr == "" {
		socialAddr = "127.0.0.1:9083"
	}
	if sc, err := grpc.NewSocialClient(socialAddr); err != nil {
		log.Printf("social gRPC unavailable (mutual-follow check disabled): %v", err)
	} else {
		defer sc.Close()
		socialCli = sc
	}

	convRepo := im.NewConversationRepository(db)
	msgRepo := im.NewMessageRepository(db)
	readRepo := im.NewReadRepository(db)
	hub := im.NewHub()
	mqPublisher := im.NewMQMessagePublisher(mq)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup

	// Meilisearch 消息搜索（可选）
	var searchClient im.MessageSearchClient
	if cfg.IM.Meilisearch.Host != "" {
		ms := im.NewMeilisearchMessageClient(
			cfg.IM.Meilisearch.Host,
			cfg.IM.Meilisearch.APIKey,
			cfg.IM.Meilisearch.Index,
		)
		_ = ms.EnsureIndex(context.Background())
		seedMessageSearchIndex(context.Background(), ms, msgRepo)
		searchClient = ms
		log.Printf("im meilisearch connected (host=%s, index=%s)", cfg.IM.Meilisearch.Host, cfg.IM.Meilisearch.Index)

		// 定时全量同步 MySQL→Meilisearch，纠正索引失败等导致的不一致
		if cfg.IM.MeilisearchSyncInterval > 0 {
			wg.Add(1)
			go runMeilisearchSync(ctx, &wg, ms, msgRepo, time.Duration(cfg.IM.MeilisearchSyncInterval)*time.Minute)
		}
	}

	svc := im.NewIMService(convRepo, msgRepo, readRepo, accountCli, socialCli, hub, mqPublisher, searchClient)
	handler := im.NewIMHandler(svc, hub)

	// 启动消息持久化消费者（异步落库）
	consumer := im.NewMessageConsumer(mq, convRepo, msgRepo, searchClient)
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer.Run(ctx)
	}()

	gin.SetMode(cfg.Server.GinMode)
	r := gin.Default()

	imGroup := r.Group("/im")
	imGroup.Use(middleware.JWTAuth(rdb, cfg.Jwt))
	{
		imGroup.POST("/send", handler.SendMessage)
		imGroup.POST("/conversations", handler.ListConversations)
		imGroup.POST("/messages", handler.ListMessages)
		imGroup.POST("/search", handler.SearchMessages)
		imGroup.POST("/read", handler.MarkRead)
	}

	// WebSocket 连接后通过首条消息鉴权（避免 token 出现在 URL）
	r.GET("/im/ws", func(c *gin.Context) {
		handler.HandleWebSocket(c, rdb, cfg.Jwt)
	})

	port := cfg.Server.IMPort
	if port == 0 {
		port = 8085
	}
	addr := fmt.Sprintf(":%d", port)
	log.Printf("IM service listening on %s", addr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err := http.ListenAndServe(addr, r); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()
	<-sigCh
	log.Printf("IM shutting down...")
	cancel()
	mq.Close()
	wg.Wait()
	log.Printf("IM stopped")
}

func openDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

func seedMessageSearchIndex(ctx context.Context, client im.MessageSearchClient, repo *im.MessageRepository) {
	syncMessageSearchIndex(ctx, client, repo)
}

// syncMessageSearchIndex 全量同步 MySQL 消息到 Meilisearch
func syncMessageSearchIndex(ctx context.Context, client im.MessageSearchClient, repo *im.MessageRepository) {
	list, err := repo.ListAllForSearch(ctx)
	if err != nil {
		log.Printf("im meilisearch sync: list messages failed: %v", err)
		return
	}
	for i := range list {
		if err := client.IndexMessage(ctx, &list[i]); err != nil {
			log.Printf("im meilisearch sync: index message %d failed: %v", list[i].ID, err)
		}
	}
	if len(list) > 0 {
		log.Printf("im meilisearch sync: indexed %d messages", len(list))
	}
}

func runMeilisearchSync(ctx context.Context, wg *sync.WaitGroup, client im.MessageSearchClient, repo *im.MessageRepository, interval time.Duration) {
	defer wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncMessageSearchIndex(ctx, client, repo)
		}
	}
}
