# Video 服务：本地存储 → 七牛云 OSS 迁移指南

本文说明如何将 Video 服务的存储从**本地磁盘**切换为**七牛云对象存储（OSS）**。

## 一、前置准备

1. **七牛云账号**：在 [七牛云控制台](https://portal.qiniu.com/) 注册并实名认证。
2. **创建存储空间（Bucket）**：
   - 进入「对象存储」→「空间管理」→「新建空间」
   - 选择区域（华东 / 华北 / 华南等）
   - 访问控制建议选择「公开空间」（视频、封面需公网访问）
3. **获取密钥**：在「个人中心」→「密钥管理」中获取 AccessKey 和 SecretKey。
4. **绑定访问域名**：为 Bucket 绑定自定义域名或使用七牛提供的测试域名（仅用于开发，生产需绑定自己的域名）。

## 二、实现七牛云存储（已完成）

项目中已提供 `internal/storage_qiniu.go`，实现了 `Storage` 接口：

- `SaveVideo`：上传视频到 `videos/{authorID}/{date}/{timestamp}.mp4`
- `SaveCover`：上传封面到 `covers/{authorID}/{date}/{timestamp}.ext`

## 三、通过配置动态切换存储

### 3.1 配置项说明

存储类型通过 `video.storage.type` 动态切换，支持 `local` 和 `qiniu`。

在 `config.yaml` 中配置：

```yaml
video:
  storage:
    type: qiniu   # 或 local
    local:
      upload_dir: .run/uploads
      static_prefix: /static
    qiniu:
      access_key: ${QINIU_ACCESS_KEY}
      secret_key: ${QINIU_SECRET_KEY}
      bucket: ${QINIU_BUCKET}
      domain: ${QINIU_DOMAIN}
      zone: z0
```

或通过环境变量：

```bash
# 切换为七牛云
export VIDEO_STORAGE_TYPE=qiniu
export QINIU_ACCESS_KEY=your_access_key
export QINIU_SECRET_KEY=your_secret_key
export QINIU_BUCKET=your-bucket-name
export QINIU_DOMAIN=https://tbrmc7lqj.hn-bkt.clouddn.com
```

### 3.2 使用七牛云

将 `video.storage.type` 设为 `qiniu` 并配置七牛云密钥即可，无需修改代码。

### 3.3 静态文件服务

- **local**：自动挂载 `r.Static(static_prefix, upload_dir)` 提供本地文件访问
- **qiniu**：视频/封面为七牛公网 URL，无需本地静态服务

## 四、URL 差异说明

| 存储方式 | 视频/封面 URL 示例 |
|---------|-------------------|
| 本地    | `http://host:8082/static/videos/1/20260311/xxx.mp4` |
| 七牛云  | `https://cdn.xxx.com/videos/1/20260311/xxx.mp4`     |

七牛云返回的 URL 已包含完整域名，`buildAbsoluteURL` 在 `video_handler.go` 中仅用于本地存储；七牛云的 `SaveVideo`/`SaveCover` 已直接返回完整 URL，无需再拼接。

## 五、依赖

七牛云 SDK 已加入 `services/video/go.mod`：

```
github.com/qiniu/go-sdk/v7
```

若在其他模块使用，需执行 `go get github.com/qiniu/go-sdk/v7`。

## 六、配置参考

相关环境变量：`VIDEO_STORAGE_TYPE`、`QINIU_ACCESS_KEY`、`QINIU_SECRET_KEY`、`QINIU_BUCKET`、`QINIU_DOMAIN`、`QINIU_ZONE`。详见 `docs/config.md`。
