package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"kairos/pkg/auth"
	"kairos/pkg/config"
	"kairos/pkg/redis"

	"github.com/gin-gonic/gin"
)

// JWTAuth 鉴权中间件：校验 JWT 并将 accountID、username 注入上下文
// 供 Video 等下游服务使用，与 Account 服务共享 Redis 存储的 Token
func JWTAuth(cache *redis.Client, cfgJwt config.JwtConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		claims, err := auth.ParseToken(tokenString, cfgJwt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		if cache == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "token store unavailable"})
			return
		}
		key := fmt.Sprintf("account:%d", claims.AccountID)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
		defer cancel()
		b, err := cache.GetBytes(ctx, key)
		if err != nil || string(b) != tokenString {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
			return
		}
		c.Set("accountID", claims.AccountID)
		c.Set("username", claims.Username)
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// GetAccountID 从上下文获取 accountID
func GetAccountID(c *gin.Context) (uint, bool) {
	v, ok := c.Get("accountID")
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}

// OptionalJWTAuth 可选鉴权：有有效 Token 时注入 accountID/username，无 Token 时继续执行
// 用于 Feed 等可匿名访问的接口，登录用户可获取 is_liked 等个性化数据
func OptionalJWTAuth(cache *redis.Client, cfgJwt config.JwtConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			c.Next()
			return
		}
		claims, err := auth.ParseToken(tokenString, cfgJwt)
		if err != nil {
			c.Next()
			return
		}
		if cache == nil {
			c.Next()
			return
		}
		key := fmt.Sprintf("account:%d", claims.AccountID)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 50*time.Millisecond)
		defer cancel()
		b, err := cache.GetBytes(ctx, key)
		if err != nil || string(b) != tokenString {
			c.Next()
			return
		}
		c.Set("accountID", claims.AccountID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
