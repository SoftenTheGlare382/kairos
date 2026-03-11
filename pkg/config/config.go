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
}

// JwtConfig JWT 配置
type JwtConfig struct {
	SecretKey string `yaml:"secret_key"`
	TokenTimeout int `yaml:"token_timeout"`
}
// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port    int    `yaml:"port"`
	GinMode string `yaml:"gin_mode"` // debug | release | test
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
	return Config{
		Server: ServerConfig{
			Port:    getEnvInt("SERVER_PORT", 8081),
			GinMode: getEnv("GIN_MODE", "debug"),
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
			DBName:   getEnv("DB_NAME", "account_db"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
	}
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
