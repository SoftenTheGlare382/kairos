# Video 服务搭建步骤说明

本文按实现顺序说明 Video 服务各组成部分的作用，便于理解和维护。

---

## 1. 共享 JWT 中间件（`pkg/middleware/jwt.go`）

**作用**：从 Account 服务抽取出的通用鉴权中间件，供 Video、Feed 等下游服务复用。

- **JWTAuth(cache, cfgJwt)**：校验请求头中的 `Authorization: Bearer <token>`
- 解析 JWT 获取 `accountID`、`username`
- 在 Redis 中校验 Token 是否有效（与 Account 登录时写入的 Token 一致，防止 Token 被吊销后仍可使用）
- 将 `accountID`、`username` 写入 Gin 上下文，供 Handler 通过 `GetAccountID(c)` 获取

**意义**：多服务共用同一套登录态，无需在 Video 服务重新实现鉴权逻辑。

---

## 2. Account 配置扩展（`pkg/config`）

**作用**：为 Video 等下游服务提供 Account 服务调用地址。

- **AccountConfig.GrpcAddr**：Account 服务的 gRPC 地址，如 `127.0.0.1:9081`（Video 通过 gRPC 调用 FindByID 等）
- 通过 `config.yaml` 或 `config.env` 的 `ACCOUNT_SERVICE_URL` 配置

**意义**：Video 需要根据 `accountID` 拉取用户信息（如 username），用于视频发布、评论展示等。

---

## 3. 实体定义（`services/video/internal/entity.go`）

**作用**：定义视频、点赞、评论的数据库表结构。

- **Video**：视频主表（标题、描述、播放地址、封面地址、作者、点赞数等）
- **Like**：点赞记录（用户-视频多对多）
- **Comment**：评论记录（视频 ID、作者、内容）

通过 GORM 标签映射到 MySQL 表，由 `AutoMigrate` 自动建表。

---

## 4. 存储接口与本地实现（`storage.go`、`storage_local.go`）

**作用**：抽象视频/封面的存储方式，方便在本地与 OSS 之间切换。

- **Storage 接口**：`SaveVideo`、`SaveCover`，接收文件流，返回可访问的 URL
- **LocalStorage**：保存到 `.run/uploads/videos/`、`.run/uploads/covers/` 目录
  - 路径格式：`{type}/{authorID}/{date}/{random}.ext`
  - 返回 URL 如 `/static/videos/1/20260311/xxx.mp4`，配合 Gin 的 `r.Static("/static", ".run/uploads")` 提供 HTTP 访问

**意义**：业务代码只依赖 `Storage` 接口，切换七牛云时只需替换实现，无需改动 Handler 或 Service。

---

## 5. 仓储层（`video_repo.go`、`like_repo.go`、`comment_repo.go`）

**作用**：封装数据库 CRUD，将 Service 与 GORM 解耦。

- **VideoRepository**：Create、GetByID、Delete、ListByAuthorID、ListByAuthorIDs、GetByIDs、UpdateLikesCount、UpdatePopularity
- **LikeRepository**：Create、Delete、IsLiked、BatchIsLiked、ListLikedVideoIDs
- **CommentRepository**：Create、Delete、GetByVideoID

**意义**：Service 只调用 Repository 方法，不直接操作 DB，便于测试和替换存储实现。

---

## 6. Account 客户端（`account_client.go`）

**作用**：通过 HTTP 调用 Account 服务的 `GET /account/findByID?id=xxx`。

- **GetByID(ctx, accountID)**：返回用户信息（至少包含 Username）

**意义**：Video 服务不直接访问 Account 数据库，通过 RPC/HTTP 获取用户信息，符合微服务拆分原则。

---

## 7. 业务服务层（`video_service.go`、`like_service.go`、`comment_service.go`）

**作用**：实现业务逻辑，校验参数、调用仓储和存储。

- **VideoService**：发布视频、删除视频、按作者列表、详情、更新点赞数等
- **LikeService**：点赞、取消点赞、是否已点赞、我点赞的视频列表
- **CommentService**：发布评论、删除评论、获取视频下所有评论

**意义**：Handler 只负责解析请求、调用 Service、返回 JSON，业务规则集中在 Service 中。

---

## 8. HTTP 处理器（`video_handler.go`、`like_handler.go`、`comment_handler.go`）

**作用**：将 HTTP 请求映射到 Service 调用，并处理错误与响应。

- **VideoHandler**：UploadVideo、UploadCover、PublishVideo、DeleteVideo、ListByAuthorID、GetDetail
- **LikeHandler**：Like、Unlike、IsLiked、ListMyLikedVideos
- **CommentHandler**：PublishComment、DeleteComment、GetAllComments

部分接口（如上传、发布、点赞、评论）需要 JWT 鉴权；部分接口（如列表、详情、评论列表）可公开。

---

## 9. main 入口（`services/video/cmd/main.go`）

**作用**：组装依赖、初始化数据库、注册路由、启动 HTTP 服务。

**步骤概览**：

1. **加载配置**：`LoadEnvFromSearchPaths` → `Load()`，从 `config.env` / `config.yaml` 读取
2. **数据库**：连接 `kairos_db`（与 Account 共用），执行 `AutoMigrate` 创建/更新表
3. **Redis**：与 Account 共用，用于 JWT 校验
4. **存储**：`NewLocalStorage`（本地）或 `NewQiniuStorage`（七牛云）
5. **仓储与服务**：创建 Repo、Service、AccountClient、Handler
6. **路由**：
   - `/static`：静态文件（本地存储时）
   - `/video/*`、`/like/*`、`/comment/*`：业务接口
   - 需鉴权接口挂在 `middleware.JWTAuth` 下
7. **监听端口**：默认 8082（通过 `VIDEO_SERVER_PORT` 或 `server.video_port` 配置）

---

## 10. 七牛云 OSS 迁移

详见 [video-oss-migration.md](./video-oss-migration.md)。

**要点**：

- 实现 `Storage` 接口的 `QiniuStorage`，已存在于 `storage_qiniu.go`
- 在 `main.go` 中根据配置（如 `QINIU_ACCESS_KEY` 是否为空）选择 `LocalStorage` 或 `QiniuStorage`
- 七牛云返回的 URL 已是完整公网地址，无需再拼接 Host

---

## 启动顺序建议

1. 启动 MySQL、Redis
2. 确保 `kairos_db` 已创建
3. 启动 Account 服务（8081）
4. 启动 Video 服务（8082）

```bash
# 在项目根目录
cd services/video
go run ./cmd
```
