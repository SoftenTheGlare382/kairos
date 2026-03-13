package config

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Jwt      JwtConfig      `yaml:"jwt"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Account  AccountConfig  `yaml:"account"`
	Video    VideoConfig    `yaml:"video"`
	Social   SocialConfig   `yaml:"social"`
	Feed     FeedConfig     `yaml:"feed"`
	Qiniu    QiniuConfig    `yaml:"qiniu"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
	Worker   WorkerConfig   `yaml:"worker"`
}

// RabbitMQConfig RabbitMQ 配置（Worker 消费、Video/Social 发布）
type RabbitMQConfig struct {
	URL string `yaml:"url"`
}

// WorkerConfig Worker 配置
type WorkerConfig struct {
	SyncIntervalMin int `yaml:"sync_interval_min"` // 全量同步间隔（分钟），0=不定时同步
}

// SocialConfig Social 服务配置
type SocialConfig struct {
	AccountGrpcAddr string `yaml:"account_grpc_addr"` // Account gRPC 地址（Social 调用）
}

// FeedConfig Feed 服务配置（依赖 Video、Social gRPC）
type FeedConfig struct {
	VideoGrpcAddr  string `yaml:"video_grpc_addr"`  // Video gRPC 地址
	SocialGrpcAddr string `yaml:"social_grpc_addr"` // Social gRPC 地址
}

// VideoConfig Video 服务配置（含 RPC 调用 Account 的地址、存储类型等）
type VideoConfig struct {
	AccountGrpcAddr string        `yaml:"account_grpc_addr"` // Account gRPC 地址（Video 调用）
	Storage         StorageConfig `yaml:"storage"`           // 存储配置（本地 / 七牛云）
}

// StorageConfig 存储配置，用于动态切换 local / qiniu
type StorageConfig struct {
	Type  string              `yaml:"type"`  // local | qiniu
	Local LocalStorageConfig  `yaml:"local"` // 本地存储配置（type=local 时生效）
	Qiniu QiniuConfig         `yaml:"qiniu"` // 七牛云配置（type=qiniu 时生效）
}

// LocalStorageConfig 本地存储配置
type LocalStorageConfig struct {
	UploadDir    string `yaml:"upload_dir"`    // 上传目录，如 .run/uploads
	StaticPrefix string `yaml:"static_prefix"` // 静态 URL 前缀，如 /static
}

// QiniuConfig 七牛云 OSS 配置（Video 服务可选）
type QiniuConfig struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	Domain    string `yaml:"domain"`
	Zone      string `yaml:"zone"` // z0华东 z1华北 z2华南 na0北美
}

// AccountConfig Account 服务地址（Video 等下游服务调用）
type AccountConfig struct {
	BaseURL  string `yaml:"base_url"`   // HTTP 地址（保留兼容）
	GrpcAddr string `yaml:"grpc_addr"`  // gRPC 地址（服务间 RPC）
}

// JwtConfig JWT 配置
type JwtConfig struct {
	SecretKey string `yaml:"secret_key"`
	TokenTimeout int `yaml:"token_timeout"`
}
// ServerConfig HTTP 服务配置
type ServerConfig struct {
	AccountPort     int    `yaml:"account_port"`      // Account HTTP 端口
	AccountGrpcPort int    `yaml:"account_grpc_port"` // Account gRPC 端口
	VideoPort       int    `yaml:"video_port"`        // Video HTTP 端口
	VideoGrpcPort   int    `yaml:"video_grpc_port"`   // Video gRPC 端口（Feed 调用）
	SocialPort      int    `yaml:"social_port"`       // Social HTTP 端口
	SocialGrpcPort  int    `yaml:"social_grpc_port"`  // Social gRPC 端口（Feed 调用）
	FeedPort        int    `yaml:"feed_port"`         // Feed HTTP 端口
	GinMode         string `yaml:"gin_mode"`          // debug | release | test
}

// DatabaseConfig MySQL 数据库配置
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// LoadEnvFile 将 config.env 中的 key=value 加载到环境变量
// 由调用方在 Load() 之前执行，使 YAML 中的 ${VAR} 能读取到这些值
// override 为 true 时覆盖已有环境变量，为 false 时不覆盖
func LoadEnvFile(path string, override bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		if key != "" && (override || os.Getenv(key) == "") {
			_ = os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

var envVarRe = regexp.MustCompile(`\$\{([^:}]+)(?::-([^}]*))?\}`)

// expandEnv 将 ${VAR} 或 ${VAR:-default} 展开为环境变量值
func expandEnv(data []byte) []byte {
	return envVarRe.ReplaceAllFunc(data, func(m []byte) []byte {
		sub := envVarRe.FindSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		key := string(sub[1])
		defaultVal := ""
		if len(sub) > 2 {
			defaultVal = string(sub[2])
		}
		if v := os.Getenv(key); v != "" {
			return []byte(v)
		}
		return []byte(defaultVal)
	})
}

// LoadFromYAML 加载 YAML 配置，支持 ${VAR} 与 ${VAR:-default} 展开（从环境变量读取）
func LoadFromYAML(yamlPath string) (Config, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return Config{}, err
	}

	expanded := expandEnv(data)

	var cfg Config
	if err := yaml.Unmarshal(expanded, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// LoadEnvFromSearchPaths 在配置搜索路径中查找 config.env 并加载到环境变量
// 由应用在 Load() 之前调用，使 YAML 中的 ${VAR} 能读到 config.env 的值
func LoadEnvFromSearchPaths(override bool) {
	dirs := searchDirs()
	for _, dir := range dirs {
		if err := LoadEnvFile(filepath.Join(dir, "config.env"), override); err == nil {
			return
		}
	}
}

func searchDirs() []string {
	if d := os.Getenv("CONFIG_DIR"); d != "" {
		abs, _ := filepath.Abs(d)
		return []string{abs}
	}
	workDir, _ := os.Getwd()
	var out []string
	for _, rel := range []string{".", "..", "../.."} {
		abs, _ := filepath.Abs(filepath.Join(workDir, rel))
		out = append(out, abs)
	}
	return out
}

// Load 加载配置：优先 YAML 文件，若 YAML 不存在或解析失败则回退到环境变量
// YAML 中的 ${VAR} 从环境变量展开，应用需在 Load 之前调用 LoadEnvFromSearchPaths 将 config.env 加载到 env
// 可通过 CONFIG_DIR 指定配置目录，默认查找 ., .., ../.. 下的 config.yaml
func Load() Config {
	for _, dir := range searchDirs() {
		yamlPath := filepath.Join(dir, "config.yaml")
		if cfg, err := LoadFromYAML(yamlPath); err == nil {
			log.Printf("loaded config from %s", yamlPath)
			return cfg
		}
	}
	log.Printf("no config file found, using environment variables")
	return LoadFromEnv()
}

// LoadFromEnv 从环境变量加载配置（兜底）
func LoadFromEnv() Config {
	cfg := Config{
		Server: ServerConfig{
			AccountPort:     getEnvInt("ACCOUNT_SERVER_PORT", 8081),
			AccountGrpcPort: getEnvInt("ACCOUNT_GRPC_PORT", 9081),
			VideoPort:       getEnvInt("VIDEO_SERVER_PORT", 8082),
			VideoGrpcPort:   getEnvInt("VIDEO_GRPC_PORT", 9082),
			SocialPort:      getEnvInt("SOCIAL_SERVER_PORT", 8083),
			SocialGrpcPort:  getEnvInt("SOCIAL_GRPC_PORT", 9083),
			FeedPort:        getEnvInt("FEED_SERVER_PORT", 8084),
			GinMode:         getEnv("GIN_MODE", "debug"),
		},
		Jwt: JwtConfig{
			SecretKey: getEnv("JWT_SECRET", "change-me-in-env"),
			TokenTimeout: getEnvInt("JWT_TOKEN_TIMEOUT", 24 * 60),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnvInt("DB_PORT", 3306),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "kairos_db"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Account: AccountConfig{
			BaseURL:  getEnv("ACCOUNT_SERVICE_URL", "http://127.0.0.1:8081"),
			GrpcAddr: getEnv("ACCOUNT_GRPC_ADDR", "127.0.0.1:9081"),
		},
		Social: SocialConfig{
			AccountGrpcAddr: getEnv("SOCIAL_ACCOUNT_GRPC_ADDR", getEnv("ACCOUNT_GRPC_ADDR", "127.0.0.1:9081")),
		},
		Feed: FeedConfig{
			VideoGrpcAddr:  getEnv("FEED_VIDEO_GRPC_ADDR", "127.0.0.1:9082"),
			SocialGrpcAddr: getEnv("FEED_SOCIAL_GRPC_ADDR", "127.0.0.1:9083"),
		},
		Video: VideoConfig{
			AccountGrpcAddr: getEnv("VIDEO_ACCOUNT_GRPC_ADDR", getEnv("ACCOUNT_GRPC_ADDR", "127.0.0.1:9081")),
			Storage: StorageConfig{
				Type: getEnv("VIDEO_STORAGE_TYPE", "local"),
				Local: LocalStorageConfig{
					UploadDir:    getEnv("VIDEO_LOCAL_UPLOAD_DIR", ".run/uploads"),
					StaticPrefix: getEnv("VIDEO_LOCAL_STATIC_PREFIX", "/static"),
				},
				Qiniu: QiniuConfig{
					AccessKey: getEnv("QINIU_ACCESS_KEY", ""),
					SecretKey: getEnv("QINIU_SECRET_KEY", ""),
					Bucket:    getEnv("QINIU_BUCKET", ""),
					Domain:    getEnv("QINIU_DOMAIN", ""),
					Zone:      getEnv("QINIU_ZONE", "z0"),
				},
			},
		},
		Qiniu: QiniuConfig{
			AccessKey: getEnv("QINIU_ACCESS_KEY", ""),
			SecretKey: getEnv("QINIU_SECRET_KEY", ""),
			Bucket:    getEnv("QINIU_BUCKET", ""),
			Domain:    getEnv("QINIU_DOMAIN", ""),
			Zone:      getEnv("QINIU_ZONE", "z0"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:5672/"),
		},
		Worker: WorkerConfig{
			SyncIntervalMin: getEnvInt("WORKER_SYNC_INTERVAL_MIN", 5),
		},
	}
	return cfg
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
