package video

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

// QiniuStorage 七牛云对象存储实现
type QiniuStorage struct {
	accessKey string
	secretKey string
	bucket    string
	domain    string // CDN 或存储域名，如 https://cdn.example.com
	mac       *qbox.Mac
	cfg       *storage.Config
}

// QiniuConfig 七牛云配置
type QiniuConfig struct {
	AccessKey string // AK
	SecretKey string // SK
	Bucket    string // 存储空间名
	Domain    string // 访问域名，如 https://cdn.xxx.com 或 https://bucket.xxx.bcebos.com
	UseHTTPS  bool   // 是否使用 HTTPS（默认 true）
	Zone      string // 区域：z0华东 z1华北 z2华南 na0北美 as0东南亚
}

// NewQiniuStorage 创建七牛云存储
func NewQiniuStorage(cfg QiniuConfig) (*QiniuStorage, error) {
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" || cfg.Domain == "" {
		return nil, fmt.Errorf("qiniu: access_key, secret_key, bucket, domain are required")
	}
	mac := qbox.NewMac(cfg.AccessKey, cfg.SecretKey)
	zone := getZone(cfg.Zone)
	cfgStore := &storage.Config{
		Zone:          zone,
		UseHTTPS:      true,
		UseCdnDomains: true,
	}
	if !cfg.UseHTTPS {
		cfgStore.UseHTTPS = false
	}
	return &QiniuStorage{
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		bucket:    cfg.Bucket,
		domain:    cfg.Domain,
		mac:       mac,
		cfg:       cfgStore,
	}, nil
}

func getZone(zone string) *storage.Zone {
	switch zone {
	case "z1":
		return &storage.ZoneHuabei
	case "z2":
		return &storage.ZoneHuanan
	case "na0":
		return &storage.ZoneBeimei
	default:
		return &storage.ZoneHuadong // z0 华东及 as0 等默认华东
	}
}

// SaveVideo 上传视频到七牛云
func (s *QiniuStorage) SaveVideo(authorID uint, filename string, body io.Reader, size int64) (string, error) {
	ext := filepath.Ext(filename)
	date := time.Now().Format("20060102")
	key := fmt.Sprintf("videos/%d/%s/%d%s", authorID, date, time.Now().UnixNano(), ext)
	return s.upload(key, body, size)
}

// SaveCover 上传封面到七牛云
func (s *QiniuStorage) SaveCover(authorID uint, filename string, body io.Reader, size int64) (string, error) {
	ext := filepath.Ext(filename)
	date := time.Now().Format("20060102")
	key := fmt.Sprintf("covers/%d/%s/%d%s", authorID, date, time.Now().UnixNano(), ext)
	return s.upload(key, body, size)
}

func (s *QiniuStorage) upload(key string, body io.Reader, size int64) (string, error) {
	putPolicy := storage.PutPolicy{Scope: s.bucket}
	upToken := putPolicy.UploadToken(s.mac)
	formUploader := storage.NewFormUploader(s.cfg)
	ret := storage.PutRet{}
	err := formUploader.Put(context.Background(), &ret, upToken, key, body, size, nil)
	if err != nil {
		return "", err
	}
	// 返回公网可访问的 URL
	url := fmt.Sprintf("%s/%s", s.domain, key)
	return url, nil
}
