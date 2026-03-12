package video

import (
	"io"
)

// Storage 存储接口：统一视频/封面上传，便于切换本地与 OSS
// 本地实现：保存到 .run/uploads/
// OSS 实现：上传至七牛云等，见 docs/oss-migration.md
type Storage interface {
	// SaveVideo 保存视频文件，返回可访问的 URL（本地为 /static/... 或 OSS 公网/CDN URL）
	SaveVideo(authorID uint, filename string, body io.Reader, size int64) (url string, err error)
	// SaveCover 保存封面图，返回可访问的 URL
	SaveCover(authorID uint, filename string, body io.Reader, size int64) (url string, err error)
}
