# 配置加载说明

所有需要读取配置的服务应统一通过 `config.Load()` 获取配置。

## 使用方式

```go
import "kairos/pkg/config"

func main() {
    // 1. 将 config.env 加载到环境变量（可选，供 YAML 中 ${VAR} 使用）
    config.LoadEnvFromSearchPaths(true)

    // 2. 获取配置
    cfg := config.Load()

    // 使用 cfg.Server、cfg.Database、cfg.Redis
}
```

## 配置结构

| 字段 | 类型 | 说明 |
|------|------|------|
| Server.Port | int | HTTP 服务端口 |
| Server.GinMode | string | Gin 模式：debug、release、test |
| Jwt.SecretKey | string | JWT 签名密钥 |
| Database.Host | string | MySQL 主机 |
| Database.Port | int | MySQL 端口 |
| Database.User | string | 数据库用户名 |
| Database.Password | string | 数据库密码 |
| Database.DBName | string | 数据库名 |
| Redis.Host | string | Redis 主机 |
| Redis.Port | int | Redis 端口 |
| Redis.Password | string | Redis 密码 |
| Redis.DB | int | Redis 库 |

## 加载优先级

1. **config.yaml**：优先加载，支持 `${VAR}`、`${VAR:-default}` 占位符
2. **环境变量**：YAML 不存在或解析失败时回退

通过 `CONFIG_DIR` 环境变量可指定配置目录。
