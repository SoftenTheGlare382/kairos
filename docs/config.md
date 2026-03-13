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
| Server.AccountPort | int | Account HTTP 端口 |
| Server.AccountGrpcPort | int | Account gRPC 端口（服务间 RPC） |
| Server.VideoPort | int | Video 服务端口 |
| Account.GrpcAddr | string | Account gRPC 地址（Video 连接用） |
| Video.AccountGrpcAddr | string | Video 调用 Account gRPC 的地址 |
| Social.AccountGrpcAddr | string | Social 调用 Account gRPC 的地址 |
| Server.SocialPort | int | Social HTTP 端口 |
| Server.SocialGrpcPort | int | Social gRPC 端口（Feed 调用） |
| Video.Storage.Type | string | 存储类型：local、qiniu |
| Video.Storage.Local.UploadDir | string | 本地存储目录 |
| Video.Storage.Local.StaticPrefix | string | 本地静态 URL 前缀 |
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
| RabbitMQ.URL | string | RabbitMQ 地址（amqp://user:pass@host:5672/），Worker 消费、Video/Social 发布 |

## 加载优先级

1. **config.yaml**：优先加载，支持 `${VAR}`、`${VAR:-default}` 占位符
2. **环境变量**：YAML 不存在或解析失败时回退

通过 `CONFIG_DIR` 环境变量可指定配置目录。
