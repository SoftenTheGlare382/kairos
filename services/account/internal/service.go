package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"kairos/pkg/auth"
	"kairos/pkg/bloomfilter"
	"kairos/pkg/config"
	"kairos/pkg/redis"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const userInfoCacheTTL = 10 * time.Minute

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
	bloom  *bloomfilter.Filter // 可选，防缓存穿透
}

// NewService 创建服务
func NewService(repo *Repository, cache *redis.Client, cfgJwt config.JwtConfig, bloom *bloomfilter.Filter) *Service {
	return &Service{repo: repo, cache: cache, cfgJwt: cfgJwt, bloom: bloom}
}

// Create 注册（bcrypt 内置盐值）
func (s *Service) Create(ctx context.Context, account *Account) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(account.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	account.Password = string(hash)
	if err := s.repo.Create(ctx, account); err != nil {
		return err
	}
	if s.bloom != nil {
		s.bloom.Add("account:id:" + strconv.FormatUint(uint64(account.ID), 10))
		s.bloom.Add("account:username:" + account.Username)
	}
	return nil
}

// Rename 重命名
func (s *Service) Rename(ctx context.Context, accountID uint, newUsername string) (string, error) {
	if newUsername == "" {
		return "", ErrNewUsernameRequired
	}
	old, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return "", err
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
	s.invalidateUserCache(ctx, old)
	if s.bloom != nil {
		s.bloom.Add("account:username:" + newUsername)
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

// FindByID 按 ID 查询（读缓存，穿透写缓存；布隆过滤器防穿透）
func (s *Service) FindByID(ctx context.Context, id uint) (*Account, error) {
	if s.bloom != nil && s.bloom.ShouldReject("account:id:"+strconv.FormatUint(uint64(id), 10)) {
		log.Printf("bloom reject: account:id:%d", id)
		return nil, gorm.ErrRecordNotFound
	}
	if s.cache != nil {
		if a := s.getUserFromCacheByID(ctx, id); a != nil {
			return a, nil
		}
	}
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a != nil && s.cache != nil {
		s.setUserCache(ctx, a)
	}
	return a, err
}

// FindByUsername 按用户名查询（读缓存，穿透写缓存；布隆过滤器防穿透）
func (s *Service) FindByUsername(ctx context.Context, username string) (*Account, error) {
	if s.bloom != nil && s.bloom.ShouldReject("account:username:"+username) {
		log.Printf("bloom reject: account:username:%s", username)
		return nil, gorm.ErrRecordNotFound
	}
	if s.cache != nil {
		if a := s.getUserFromCacheByUsername(ctx, username); a != nil {
			return a, nil
		}
	}
	a, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if a != nil && s.cache != nil {
		s.setUserCache(ctx, a)
	}
	return a, err
}

// FindByIDs 批量按 ID 查询（布隆过滤不存在的 id，再读缓存/DB）
func (s *Service) FindByIDs(ctx context.Context, ids []uint) ([]Account, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	toFetch := make([]uint, 0, len(ids))
	if s.bloom != nil {
		for _, id := range ids {
			if s.bloom.ShouldReject("account:id:" + strconv.FormatUint(uint64(id), 10)) {
				log.Printf("bloom reject: account:id:%d (FindByIDs)", id)
			} else {
				toFetch = append(toFetch, id)
			}
		}
	} else {
		toFetch = ids
	}
	if len(toFetch) == 0 {
		return nil, nil
	}
	cached := make(map[uint]Account)
	miss := make([]uint, 0, len(toFetch))
	if s.cache != nil {
		for _, id := range toFetch {
			if a := s.getUserFromCacheByID(ctx, id); a != nil {
				cached[id] = *a
			} else {
				miss = append(miss, id)
			}
		}
	} else {
		miss = toFetch
	}
		if len(miss) > 0 {
		list, err := s.repo.FindByIDs(ctx, miss)
		if err != nil {
			return nil, err
		}
		for _, a := range list {
			cached[a.ID] = a
			if s.cache != nil {
				s.setUserCache(ctx, &a)
			}
		}
	} else if s.cache != nil && len(miss) == 0 {
		log.Printf("cache hit: account:FindByIDs (all %d ids)", len(toFetch))
	}
	out := make([]Account, 0, len(ids))
	for _, id := range ids {
		if a, ok := cached[id]; ok {
			out = append(out, a)
		}
	}
	return out, nil
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

// Cancel 注销账户（软删除，清除 Redis Token 与用户缓存）
func (s *Service) Cancel(ctx context.Context, accountID uint) error {
	old, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return err
	}
	_ = s.repo.UpdateLastLogoutAt(ctx, accountID, time.Now())
	s.delCache(ctx, accountID)
	s.invalidateUserCache(ctx, old)
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
	if err := s.cache.SetBytes(ctx2, key, []byte(token), redis.TTLWithJitter(24*time.Hour, 0.2)); err != nil {
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

// 用户信息缓存 key：account:info:{id}, account:username:{username}
func userInfoKey(id uint) string { return fmt.Sprintf("account:info:%d", id) }
func userUsernameKey(username string) string { return fmt.Sprintf("account:username:%s", username) }

func (s *Service) setUserCache(ctx context.Context, a *Account) {
	if s.cache == nil || a == nil {
		return
	}
	b, err := json.Marshal(a)
	if err != nil {
		return
	}
	ctx2, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_ = s.cache.SetBytes(ctx2, userInfoKey(a.ID), b, redis.TTLWithJitter(userInfoCacheTTL, 0.2))
	_ = s.cache.SetBytes(ctx2, userUsernameKey(a.Username), b, redis.TTLWithJitter(userInfoCacheTTL, 0.2))
}

func (s *Service) getUserFromCacheByID(ctx context.Context, id uint) *Account {
	if s.cache == nil {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	b, err := s.cache.GetBytes(ctx2, userInfoKey(id))
	if err != nil || len(b) == 0 {
		return nil
	}
	var a Account
	if json.Unmarshal(b, &a) != nil {
		return nil
	}
	log.Printf("cache hit: account:info:%d", id)
	return &a
}

func (s *Service) getUserFromCacheByUsername(ctx context.Context, username string) *Account {
	if s.cache == nil {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	b, err := s.cache.GetBytes(ctx2, userUsernameKey(username))
	if err != nil || len(b) == 0 {
		return nil
	}
	var a Account
	if json.Unmarshal(b, &a) != nil {
		return nil
	}
	log.Printf("cache hit: account:username:%s", username)
	return &a
}

func (s *Service) invalidateUserCache(ctx context.Context, a *Account) {
	if s.cache == nil || a == nil {
		return
	}
	ctx2, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_ = s.cache.Del(ctx2, userInfoKey(a.ID))
	_ = s.cache.Del(ctx2, userUsernameKey(a.Username))
}
