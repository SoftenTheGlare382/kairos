package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"kairos/api/videopb"
	"kairos/pkg/bloomfilter"
	"kairos/pkg/config"
	"kairos/pkg/events"
	grpcpkg "kairos/pkg/grpc"
	"kairos/pkg/middleware"
	"kairos/pkg/rabbitmq"
	"kairos/pkg/redis"
	video "kairos/services/video/internal"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	config.LoadEnvFromSearchPaths(true)
	cfg := config.Load()

	// 与 Account 服务共用 kairos_db，不同表
	db, err := openDB(cfg.Database)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := db.AutoMigrate(&video.Video{}, &video.Like{}, &video.Comment{}, &video.PlayRecord{}, &video.Favorite{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// Redis：与 Account 共享，用于 JWT 校验
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

	// 存储：根据 video.storage.type 动态切换 local / qiniu
	var storage video.Storage
	switch cfg.Video.Storage.Type {
	case "qiniu":
		qiniu := cfg.Video.Storage.Qiniu
		if qiniu.AccessKey == "" {
			qiniu = cfg.Qiniu // 兜底：使用顶层 qiniu 配置
		}
		if qiniu.AccessKey == "" || qiniu.SecretKey == "" || qiniu.Bucket == "" || qiniu.Domain == "" {
			log.Fatalf("video.storage.type=qiniu 时需配置 access_key, secret_key, bucket, domain")
		}
		var err error
		storage, err = video.NewQiniuStorage(video.QiniuConfig{
			AccessKey: qiniu.AccessKey,
			SecretKey: qiniu.SecretKey,
			Bucket:    qiniu.Bucket,
			Domain:    qiniu.Domain,
			Zone:      qiniu.Zone,
		})
		if err != nil {
			log.Fatalf("qiniu storage: %v", err)
		}
		log.Printf("using qiniu OSS storage")
	default:
		local := cfg.Video.Storage.Local
		if local.UploadDir == "" {
			local.UploadDir = ".run/uploads"
		}
		if local.StaticPrefix == "" {
			local.StaticPrefix = "/static"
		}
		storage = video.NewLocalStorage(local.UploadDir, local.StaticPrefix)
		log.Printf("using local storage (dir=%s)", local.UploadDir)
	}

	// RabbitMQ（可选，用于发布 like/comment/popularity 事件给 Worker）
	var publisher *events.Publisher
	if cfg.RabbitMQ.URL != "" {
		if mq, err := rabbitmq.New(cfg.RabbitMQ.URL); err != nil {
			log.Printf("rabbitmq connect failed (worker events disabled): %v", err)
		} else {
			defer mq.Close()
			publisher = events.NewPublisher(mq)
			log.Printf("rabbitmq connected, publishing like/comment/popularity events")
		}
	}

	// 仓储与服务
	videoRepo := video.NewVideoRepository(db)
	likeRepo := video.NewLikeRepository(db)
	commentRepo := video.NewCommentRepository(db)
	playRecordRepo := video.NewPlayRecordRepository(db)
	favoriteRepo := video.NewFavoriteRepository(db)

	videoBloom := bloomfilter.New(10_000_000, 0.01)
	seedVideoBloom(context.Background(), videoBloom, videoRepo)
	videoBloom.SetReady()

	// 定时重建布隆过滤器，清除已删除视频
	go runVideoBloomRebuild(videoRepo, videoBloom, 6*time.Hour)

	// Meilisearch 搜索（可选）
	var searchClient video.SearchClient
	if cfg.Video.Meilisearch.Host != "" {
		ms := video.NewMeilisearchClient(
			cfg.Video.Meilisearch.Host,
			cfg.Video.Meilisearch.APIKey,
			cfg.Video.Meilisearch.Index,
		)
		_ = ms.EnsureIndex(context.Background())
		seedVideoSearchIndex(context.Background(), ms, videoRepo)
		searchClient = ms
		log.Printf("meilisearch connected (host=%s, index=%s)", cfg.Video.Meilisearch.Host, cfg.Video.Meilisearch.Index)
	}

	videoSvc := video.NewVideoService(videoRepo, storage, rdb, videoBloom, searchClient)
	likeSvc := video.NewLikeService(db, likeRepo, videoRepo, publisher, rdb)
	commentSvc := video.NewCommentService(commentRepo, videoRepo, publisher, rdb, videoBloom)
	playSvc := video.NewPlayService(db, playRecordRepo, videoRepo, publisher, publisher)
	favoriteSvc := video.NewFavoriteService(db, favoriteRepo, videoRepo, publisher)

	grpcAddr := cfg.Video.AccountGrpcAddr
	if grpcAddr == "" {
		grpcAddr = cfg.Account.GrpcAddr
	}
	accountCli, err := grpcpkg.NewAccountClient(grpcAddr)
	if err != nil {
		log.Fatalf("account gRPC client: %v", err)
	}
	defer accountCli.Close()

	videoHandler := video.NewVideoHandler(videoSvc, accountCli, storage)
	likeHandler := video.NewLikeHandler(likeSvc)
	commentHandler := video.NewCommentHandler(commentSvc, accountCli)
	playHandler := video.NewPlayHandler(playSvc, accountCli)
	favoriteHandler := video.NewFavoriteHandler(favoriteSvc)

	// 启动 Video gRPC 服务（供 Feed 调用）
	grpcPort := cfg.Server.VideoGrpcPort
	if grpcPort == 0 {
		grpcPort = 9082
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("video grpc listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	videopb.RegisterVideoServiceServer(grpcServer, video.NewGrpcVideoServer(videoRepo, likeRepo, rdb))
	reflection.Register(grpcServer)
	go func() {
		log.Printf("Video gRPC listening on :%d", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("video grpc serve: %v", err)
		}
	}()

	gin.SetMode(cfg.Server.GinMode)
	r := gin.Default()

	// 本地存储时：提供静态目录访问上传的文件；七牛云时无需
	if ls, ok := storage.(*video.LocalStorage); ok {
		r.Static(ls.StaticURL, ls.RootDir)
	}

	// 公开接口
	videoGroup := r.Group("/video")
	{
		videoGroup.POST("/listByAuthorID", videoHandler.ListByAuthorID)
		videoGroup.POST("/getDetail", videoHandler.GetDetail)
		videoGroup.POST("/search", videoHandler.Search)
	}
	commentGroup := r.Group("/comment")
	{
		commentGroup.POST("/listAll", commentHandler.GetAllComments)
	}

	// 需鉴权接口
	protected := r.Group("")
	protected.Use(middleware.JWTAuth(rdb, cfg.Jwt))
	{
		protected.POST("/video/uploadVideo", videoHandler.UploadVideo)
		protected.POST("/video/uploadCover", videoHandler.UploadCover)
		protected.POST("/video/publish", videoHandler.PublishVideo)
		protected.POST("/video/delete", videoHandler.DeleteVideo)
		protected.POST("/video/recordPlay", playHandler.RecordPlay)
		protected.POST("/video/listPlayRecords", playHandler.ListPlayRecords)

		protected.POST("/video/favorite", favoriteHandler.Favorite)
		protected.POST("/video/unfavorite", favoriteHandler.Unfavorite)
		protected.POST("/video/isFavorited", favoriteHandler.IsFavorited)
		protected.POST("/video/listMyFavoritedVideos", favoriteHandler.ListMyFavoritedVideos)

		protected.POST("/like/like", likeHandler.Like)
		protected.POST("/like/unlike", likeHandler.Unlike)
		protected.POST("/like/isLiked", likeHandler.IsLiked)
		protected.POST("/like/listMyLikedVideos", likeHandler.ListMyLikedVideos)

		protected.POST("/comment/publish", commentHandler.PublishComment)
		protected.POST("/comment/delete", commentHandler.DeleteComment)
	}

	port := cfg.Server.VideoPort
	if port == 0 {
		port = 8082
	}
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Video service listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func seedVideoSearchIndex(ctx context.Context, searchClient video.SearchClient, repo *video.VideoRepository) {
	list, err := repo.ListAllForSearch(ctx)
	if err != nil {
		return
	}
	for _, v := range list {
		_ = searchClient.IndexVideo(ctx, &v)
	}
	if len(list) > 0 {
		log.Printf("meilisearch seed: indexed %d videos", len(list))
	}
}

func seedVideoBloom(ctx context.Context, bloom *bloomfilter.Filter, repo *video.VideoRepository) {
	keys := collectVideoBloomKeys(ctx, repo)
	if len(keys) > 0 {
		bloom.Rebuild(keys)
	}
}

func collectVideoBloomKeys(ctx context.Context, repo *video.VideoRepository) []string {
	ids, err := repo.ListAllIDs(ctx)
	if err != nil {
		return nil
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, "video:"+strconv.FormatUint(uint64(id), 10))
	}
	return keys
}

func runVideoBloomRebuild(repo *video.VideoRepository, bloom *bloomfilter.Filter, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		keys := collectVideoBloomKeys(ctx, repo)
		cancel()
		if len(keys) > 0 {
			bloom.Rebuild(keys)
			log.Printf("video bloom filter rebuilt with %d keys", len(keys))
		}
	}
}

func openDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
