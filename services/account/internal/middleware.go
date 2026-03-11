package account

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"kairos/pkg/auth"
	"kairos/pkg/redis"

	"github.com/gin-gonic/gin"
	"kairos/pkg/config"
)

// JWTAuth 鉴权中间件：校验 JWT 并将 accountID、username 注入上下文
// Token 仅从 Redis 校验，cache 不可为空
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

// extractToken 提取 Token
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
