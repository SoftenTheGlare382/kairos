package video

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"crypto/rand"
	"encoding/hex"
)

// LocalStorage 本地磁盘存储，文件保存在 .run/uploads/
type LocalStorage struct {
	RootDir   string // 如 .run/uploads
	StaticURL string // 如 /static，用于拼接返回给前端的 URL
}

// NewLocalStorage 创建本地存储
func NewLocalStorage(rootDir, staticURL string) *LocalStorage {
	if rootDir == "" {
		rootDir = ".run/uploads"
	}
	if staticURL == "" {
		staticURL = "/static"
	}
	return &LocalStorage{RootDir: rootDir, StaticURL: staticURL}
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SaveVideo 保存视频到 videos/{authorID}/{date}/{random}.ext
func (s *LocalStorage) SaveVideo(authorID uint, filename string, body io.Reader, size int64) (string, error) {
	ext := filepath.Ext(filename)
	date := time.Now().Format("20060102")
	relPath := filepath.Join("videos", fmt.Sprintf("%d", authorID), date, randHex(16)+ext)
	return s.save(relPath, body, size)
}

// SaveCover 保存封面对 covers/{authorID}/{date}/{random}.ext
func (s *LocalStorage) SaveCover(authorID uint, filename string, body io.Reader, size int64) (string, error) {
	ext := filepath.Ext(filename)
	date := time.Now().Format("20060102")
	relPath := filepath.Join("covers", fmt.Sprintf("%d", authorID), date, randHex(16)+ext)
	return s.save(relPath, body, size)
}

func (s *LocalStorage) save(relPath string, body io.Reader, size int64) (string, error) {
	absPath := filepath.Join(s.RootDir, relPath)
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(absPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, body); err != nil {
		os.Remove(absPath)
		return "", err
	}
	// 返回 URL 路径，前端需配合 Gin Static 或 Nginx 提供访问
	urlPath := s.StaticURL + "/" + filepath.ToSlash(relPath)
	return urlPath, nil
}
