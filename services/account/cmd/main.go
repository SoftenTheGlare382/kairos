package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"kairos/pkg/config"
	"kairos/pkg/redis"
	account "kairos/services/account/internal"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
	protected.Use(account.JWTAuth(cache, cfg.Jwt))
	{
		protected.POST("/logout", handler.Logout)
		protected.POST("/rename", handler.Rename)
		protected.POST("/cancel", handler.Cancel)
		accountGroup.POST("/changePassword", handler.ChangePassword)
		accountGroup.POST("/findByID", handler.FindByID)
		accountGroup.POST("/findByUsername", handler.FindByUsername)
	}

	// 内部接口（供 Gateway 校验 Token）
	internalGroup := r.Group("/internal")
	{
		internalGroup.POST("/validate", handler.Validate)
	}

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Account service listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func openDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	return gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
