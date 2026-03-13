package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"kairos/api/socialpb"
	"kairos/pkg/config"
	"kairos/pkg/events"
	grpcpkg "kairos/pkg/grpc"
	"kairos/pkg/middleware"
	"kairos/pkg/rabbitmq"
	"kairos/pkg/redis"
	social "kairos/services/social/internal"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	if err := db.AutoMigrate(&social.Follow{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	rdb := redis.New(redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()); err != nil {
		log.Fatalf("redis required for auth: %v", err)
	}
	defer rdb.Close()

	socialRepo := social.NewSocialRepository(db)

	// RabbitMQ（可选，用于发布 social 事件给 Worker）
	var publisher *events.Publisher
	if cfg.RabbitMQ.URL != "" {
		if mq, err := rabbitmq.New(cfg.RabbitMQ.URL); err != nil {
			log.Printf("rabbitmq connect failed (worker events disabled): %v", err)
		} else {
			defer mq.Close()
			publisher = events.NewPublisher(mq)
			log.Printf("rabbitmq connected, publishing social events")
		}
	}

	grpcAddr := cfg.Social.AccountGrpcAddr
	if grpcAddr == "" {
		grpcAddr = cfg.Account.GrpcAddr
	}
	accountCli, err := grpcpkg.NewAccountClient(grpcAddr)
	if err != nil {
		log.Fatalf("account gRPC client: %v", err)
	}
	defer accountCli.Close()

	socialSvc := social.NewSocialService(socialRepo, accountCli, publisher)
	socialHandler := social.NewSocialHandler(socialSvc)

	// 启动 gRPC 服务（供 Feed 调用 GetFollowingIDs）
	grpcPort := cfg.Server.SocialGrpcPort
	if grpcPort == 0 {
		grpcPort = 9083
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	socialpb.RegisterSocialServiceServer(grpcServer, social.NewGrpcSocialServer(socialRepo))
	reflection.Register(grpcServer)
	go func() {
		log.Printf("Social gRPC listening on :%d", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	gin.SetMode(cfg.Server.GinMode)
	r := gin.Default()

	protected := r.Group("/social")
	protected.Use(middleware.JWTAuth(rdb, cfg.Jwt))
	{
		protected.POST("/follow", socialHandler.Follow)
		protected.POST("/unfollow", socialHandler.Unfollow)
		protected.POST("/followers", socialHandler.Followers)
		protected.POST("/following", socialHandler.Following)
	}

	port := cfg.Server.SocialPort
	if port == 0 {
		port = 8083
	}
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Social HTTP listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func openDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
