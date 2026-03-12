package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"kairos/pkg/config"
	"kairos/pkg/redis"
	"kairos/pkg/middleware"
	video "kairos/services/video/internal"

	"github.com/gin-gonic/gin"
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

	if err := db.AutoMigrate(&video.Video{}, &video.Like{}, &video.Comment{}); err != nil {
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

	// 仓储与服务
	videoRepo := video.NewVideoRepository(db)
	likeRepo := video.NewLikeRepository(db)
	commentRepo := video.NewCommentRepository(db)

	videoSvc := video.NewVideoService(videoRepo, storage)
	likeSvc := video.NewLikeService(db, likeRepo, videoRepo)
	commentSvc := video.NewCommentService(commentRepo, videoRepo)

	var accountCli *video.AccountClient
	grpcAddr := cfg.Video.AccountGrpcAddr
	if grpcAddr == "" {
		grpcAddr = cfg.Account.GrpcAddr
	}
	accountCli, err = video.NewAccountClient(grpcAddr)
	if err != nil {
		log.Fatalf("account gRPC client: %v", err)
	}
	defer accountCli.Close()

	videoHandler := video.NewVideoHandler(videoSvc, accountCli, storage)
	likeHandler := video.NewLikeHandler(likeSvc)
	commentHandler := video.NewCommentHandler(commentSvc, accountCli)

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

func openDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
