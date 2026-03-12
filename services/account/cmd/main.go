package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"kairos/pkg/config"
	"kairos/pkg/redis"
	account "kairos/services/account/internal"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	accountpb "kairos/api/accountpb"
	"kairos/pkg/middleware"
)

func main() {
	// 配置加载：见 docs/config.md
	config.LoadEnvFromSearchPaths(true)
	cfg := config.Load()

	db, err := openDB(cfg.Database)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := db.AutoMigrate(&account.Account{}); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	// Token 仅存 Redis，Redis 为必选
	rdb := redis.New(redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()); err != nil {
		log.Fatalf("redis required for token storage: %v", err)
	}
	defer rdb.Close()
	cache := rdb

	repo := account.NewRepository(db)
	svc := account.NewService(repo, cache, cfg.Jwt)
	handler := account.NewHandler(svc)

	gin.SetMode(cfg.Server.GinMode)
	r := gin.Default()

	// 公开接口
	accountGroup := r.Group("/account")
	{
		accountGroup.POST("/register", handler.Register)
		accountGroup.POST("/login", handler.Login)
	}

	// 需鉴权接口
	protected := accountGroup.Group("")
	protected.Use(middleware.JWTAuth(cache, cfg.Jwt))
	{
		protected.POST("/logout", handler.Logout)
		protected.POST("/rename", handler.Rename)
		protected.POST("/cancel", handler.Cancel)
		protected.POST("/changePassword", handler.ChangePassword)
		protected.POST("/findByID", handler.FindByID)
		protected.POST("/findByUsername", handler.FindByUsername)
	}

	// 内部接口（供 Gateway 校验 Token）
	internalGroup := r.Group("/internal")
	{
		internalGroup.POST("/validate", handler.Validate)
	}

	// 启动 gRPC 服务（供 Video 等下游服务 RPC 调用）
	grpcPort := cfg.Server.AccountGrpcPort
	if grpcPort == 0 {
		grpcPort = 9081
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	accountpb.RegisterAccountServiceServer(grpcServer, account.NewGrpcAccountServer(svc))
	reflection.Register(grpcServer)
	go func() {
		log.Printf("Account gRPC listening on :%d", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// HTTP 服务
	addr := fmt.Sprintf(":%d", cfg.Server.AccountPort)
	log.Printf("Account HTTP listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func openDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
