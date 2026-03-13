package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"kairos/pkg/config"
	"kairos/pkg/middleware"
	"kairos/pkg/redis"
	feed "kairos/services/feed/internal"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnvFromSearchPaths(true)
	cfg := config.Load()

	// Feed 无独立 DB，依赖 Redis（鉴权）和 gRPC
	rdb := redis.New(redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()); err != nil {
		log.Printf("redis ping: %v (optional for feed)", err)
	}
	defer rdb.Close()

	videoAddr := cfg.Feed.VideoGrpcAddr
	if videoAddr == "" {
		videoAddr = "127.0.0.1:9082"
	}
	socialAddr := cfg.Feed.SocialGrpcAddr
	if socialAddr == "" {
		socialAddr = "127.0.0.1:9083"
	}

	videoCli, err := feed.NewVideoClient(videoAddr)
	if err != nil {
		log.Fatalf("video gRPC client: %v", err)
	}
	defer videoCli.Close()

	socialCli, err := feed.NewSocialClient(socialAddr)
	if err != nil {
		log.Fatalf("social gRPC client: %v", err)
	}
	defer socialCli.Close()

	svc := feed.NewService(videoCli, socialCli)
	handler := feed.NewHandler(svc)

	gin.SetMode(cfg.Server.GinMode)
	r := gin.Default()

	// 最新流、热度流、关注流：均需鉴权
	feedGroup := r.Group("/feed")
	feedGroup.Use(middleware.JWTAuth(rdb, cfg.Jwt))
	{
		feedGroup.POST("/listLatest", handler.ListLatest)
		feedGroup.POST("/listByPopularity", handler.ListByPopularity)
		feedGroup.POST("/listByFollowing", handler.ListByFollowing)
	}

	port := cfg.Server.FeedPort
	if port == 0 {
		port = 8084
	}
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Feed HTTP listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
