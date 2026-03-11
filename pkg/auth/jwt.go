package auth

import (
	"errors"

	"time"

	"github.com/golang-jwt/jwt/v5"
	"kairos/pkg/config"
)

func jwtSecret(cfgJwt config.JwtConfig) []byte {
	secret := cfgJwt.SecretKey
	if secret == "" {
		secret = "change-me-in-env"
	}
	return []byte(secret)
}

// Claims JWT 声明
type Claims struct {
	AccountID uint   `json:"account_id"`
	Username  string `json:"username"`
	jwt.RegisteredClaims//
}

// GenerateToken 生成 JWT Token
func GenerateToken(accountID uint, username string,cfgJwt config.JwtConfig) (string, error) {
	now := time.Now()
	claims := Claims{
		AccountID: accountID,
		Username:  username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add( time.Duration(cfgJwt.TokenTimeout) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret(cfgJwt))
}

// ParseToken 解析并校验 JWT Token
func ParseToken(tokenString string,cfg config.JwtConfig) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return jwtSecret(cfg), nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
