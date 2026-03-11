package account

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"kairos/pkg/auth"
	"kairos/pkg/config"
	"kairos/pkg/redis"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken    = errors.New("username already exists")
	ErrNewUsernameRequired = errors.New("new_username is required")
	ErrTokenRevoked     = errors.New("token has been revoked")
)

// Service 账户服务
type Service struct {
	repo   *Repository
	cache  *redis.Client
	cfgJwt config.JwtConfig
}

// NewService 创建服务
func NewService(repo *Repository, cache *redis.Client, cfgJwt config.JwtConfig) *Service {
	return &Service{repo: repo, cache: cache, cfgJwt: cfgJwt}
}

// Create 注册（bcrypt 内置盐值）
func (s *Service) Create(ctx context.Context, account *Account) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	account.Password = string(hash)
	return s.repo.Create(ctx, account)
}

// Rename 重命名
func (s *Service) Rename(ctx context.Context, accountID uint, newUsername string) (string, error) {
	if newUsername == "" {
		return "", ErrNewUsernameRequired
	}
	token, err := auth.GenerateToken(accountID, newUsername, s.cfgJwt)
	if err != nil {
		return "", err
	}
	if err := s.repo.Rename(ctx, accountID, newUsername); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return "", ErrUsernameTaken
		}
		return "", err
	}
	s.setCache(ctx, accountID, token)
	return token, nil
}

// ChangePassword 修改密码
func (s *Service) ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error {
	account, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(oldPassword)); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, account.ID, string(hash)); err != nil {
		return err
	}
	return s.Logout(ctx, account.ID)
}

// FindByID 按 ID 查询
func (s *Service) FindByID(ctx context.Context, id uint) (*Account, error) {
	return s.repo.FindByID(ctx, id)
}

// FindByUsername 按用户名查询
func (s *Service) FindByUsername(ctx context.Context, username string) (*Account, error) {
	return s.repo.FindByUsername(ctx, username)
}

// Login 登录（软删除账户不会查到）
func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	account, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password)); err != nil {
		return "", err
	}
	token, err := auth.GenerateToken(account.ID, account.Username, s.cfgJwt)
	if err != nil {
		return "", err
	}
	s.setCache(ctx, account.ID, token)
	_ = s.repo.UpdateLastLoginAt(ctx, account.ID, time.Now())
	return token, nil
}

// Logout 登出
func (s *Service) Logout(ctx context.Context, accountID uint) error {
	_ = s.repo.UpdateLastLogoutAt(ctx, accountID, time.Now())
	s.delCache(ctx, accountID)
	return nil
}

// Cancel 注销账户（软删除，清除 Redis Token）
func (s *Service) Cancel(ctx context.Context, accountID uint) error {
	if _, err := s.repo.FindByID(ctx, accountID); err != nil {
		return err
	}
	_ = s.repo.UpdateLastLogoutAt(ctx, accountID, time.Now())
	s.delCache(ctx, accountID)
	return s.repo.Delete(ctx, accountID)
}

// ValidateToken 校验 Token（供 Gateway 调用），返回 account_id、username
// Token 仅校验 Redis，不查 MySQL
func (s *Service) ValidateToken(ctx context.Context, tokenString string) (uint, string, error) {
	claims, err := auth.ParseToken(tokenString, s.cfgJwt)
	if err != nil {
		return 0, "", err
	}
	if s.cache == nil {
		return 0, "", ErrTokenRevoked
	}
	key := fmt.Sprintf("account:%d", claims.AccountID)
	ctx2, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	b, err := s.cache.GetBytes(ctx2, key)
	if err != nil || string(b) != tokenString {
		return 0, "", ErrTokenRevoked
	}
	return claims.AccountID, claims.Username, nil
}

func (s *Service) setCache(ctx context.Context, accountID uint, token string) {
	if s.cache == nil {
		return
	}
	ctx2, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	key := fmt.Sprintf("account:%d", accountID)
	if err := s.cache.SetBytes(ctx2, key, []byte(token), 24*time.Hour); err != nil {
		log.Printf("cache set failed: %v", err)
	}
}

func (s *Service) delCache(ctx context.Context, accountID uint) {
	if s.cache == nil {
		return
	}
	ctx2, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	key := fmt.Sprintf("account:%d", accountID)
	if err := s.cache.Del(ctx2, key); err != nil {
		log.Printf("cache del failed: %v", err)
	}
}
