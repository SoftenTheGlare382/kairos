# Video 服务 API 接口文档

**基础路径**：`/video`、`/comment`、`/like`  
**端口**：8082（可通过 config 配置）

**热度公式**：`popularity` = 点赞×20% + 评论×40% + 收藏×30% + 观看×10%（整数权重 2:4:3:1）

---

## 一、视频接口

### 1. 按作者列出视频（公开）

**POST** `/video/listByAuthorID`

### 请求体

```json
{
  "author_id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| author_id | number | 是 | 作者账户 ID |

### 响应

**200 OK**
```json
[
  {
    "id": 1,
    "author_id": 1,
    "username": "string",
    "title": "string",
    "description": "string",
    "play_url": "string",
    "cover_url": "string",
    "created_at": "2026-03-11T10:00:00Z",
    "likes_count": 0,
    "popularity": 0
  }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | number | 视频 ID |
| author_id | number | 作者账户 ID |
| username | string | 作者用户名 |
| title | string | 标题 |
| description | string | 描述 |
| play_url | string | 播放地址 |
| cover_url | string | 封面地址 |
| created_at | string | 创建时间，ISO8601 |
| likes_count | number | 点赞数 |
| popularity | number | 热度值 |
| play_count | number | 播放数 |
| favorites_count | number | 收藏数 |

**400 Bad Request**
```json
{
  "error": "请求解析错误信息"
}
```

---

### 2. 视频详情（公开）

**POST** `/video/getDetail`

### 请求体

```json
{
  "id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | number | 是 | 视频 ID |

### 响应

**200 OK**
```json
{
  "id": 1,
  "author_id": 1,
  "username": "string",
  "title": "string",
  "description": "string",
  "play_url": "string",
  "cover_url": "string",
  "created_at": "2026-03-11T10:00:00Z",
  "likes_count": 0,
  "popularity": 0,
  "play_count": 0,
  "favorites_count": 0
}
```

**404 Not Found**
```json
{
  "error": "video not found"
}
```

---

### 3. 上传视频文件

**POST** `/video/uploadVideo`

**需要鉴权**：`Authorization: Bearer <token>`

**Content-Type**：`multipart/form-data`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 视频文件，仅支持 .mp4，最大 200MB |

### 响应

**200 OK**
```json
{
  "url": "string",
  "play_url": "string"
}
```

> 返回的 play_url 用于后续发布视频时传入 `publish` 接口。

**400 Bad Request**
```json
{
  "error": "missing file"
}
```
或
```json
{
  "error": "invalid file size"
}
```
或
```json
{
  "error": "only .mp4 is allowed"
}
```

---

### 4. 上传封面

**POST** `/video/uploadCover`

**需要鉴权**：`Authorization: Bearer <token>`

**Content-Type**：`multipart/form-data`

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 封面图片，支持 .jpg/.jpeg/.png/.webp，最大 10MB |

### 响应

**200 OK**
```json
{
  "url": "string",
  "cover_url": "string"
}
```

**400 Bad Request**
```json
{
  "error": "missing file"
}
```
或
```json
{
  "error": "invalid file size"
}
```
或
```json
{
  "error": "only .jpg/.jpeg/.png/.webp is allowed"
}
```

---

### 5. 发布视频

**POST** `/video/publish`

**需要鉴权**：`Authorization: Bearer <token>`

通常先调用 `uploadVideo`、`uploadCover` 获取 play_url、cover_url，再调用本接口发布。

### 请求体

```json
{
  "title": "string",
  "description": "string",
  "play_url": "string",
  "cover_url": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 标题 |
| description | string | 否 | 描述 |
| play_url | string | 是 | 播放地址（来自 uploadVideo） |
| cover_url | string | 是 | 封面地址（来自 uploadCover） |

### 响应

**200 OK**
```json
{
  "id": 1,
  "author_id": 1,
  "username": "string",
  "title": "string",
  "description": "string",
  "play_url": "string",
  "cover_url": "string",
  "created_at": "2026-03-11T10:00:00Z",
  "likes_count": 0,
  "popularity": 0
}
```

**400 Bad Request**
```json
{
  "error": "title is required"
}
```
或
```json
{
  "error": "play_url is required"
}
```
或
```json
{
  "error": "cover_url is required"
}
```
或
```json
{
  "error": "failed to get user: ..."
}
```

---

### 6. 删除视频

**POST** `/video/delete`

**需要鉴权**：`Authorization: Bearer <token>`

仅作者可删除自己的视频。

### 请求体

```json
{
  "id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | number | 是 | 视频 ID |

### 响应

**200 OK**
```json
{
  "message": "video deleted"
}
```

**400 Bad Request**
```json
{
  "error": "video not found"
}
```
或
```json
{
  "error": "unauthorized"
}
```

---

### 7. 记录播放

**POST** `/video/recordPlay`

**需要鉴权**：`Authorization: Bearer <token>`

前端在用户播放视频时调用，用于统计播放量与「谁播放了几次、最近播放时间」。

### 请求体

```json
{
  "video_id": 1
}
```

### 响应

**200 OK**
```json
{
  "message": "play recorded"
}
```

---

### 8. 播放记录列表（仅作者可查）

**POST** `/video/listPlayRecords`

**需要鉴权**：`Authorization: Bearer <token>`

仅视频作者可查询该视频的播放记录，包含谁播放了几次、最近播放时间。

### 请求体

```json
{
  "video_id": 1,
  "limit": 20,
  "offset": 0
}
```

### 响应

**200 OK**
```json
[
  {
    "account_id": 2,
    "username": "user2",
    "play_count": 5,
    "last_play_at": "2026-03-12T10:30:00Z"
  }
]
```

---

### 9. 收藏 / 取消收藏 / 是否已收藏 / 我收藏的视频

**POST** `/video/favorite` — 收藏

**POST** `/video/unfavorite` — 取消收藏

**POST** `/video/isFavorited` — 是否已收藏（未登录返回 false）

**POST** `/video/listMyFavoritedVideos` — 我收藏的视频列表

请求体与点赞接口类似，`video_id` 必填。响应格式与点赞接口类似。

---

## 二、点赞接口

### 1. 点赞

**POST** `/like/like`

**需要鉴权**：`Authorization: Bearer <token>`

### 请求体

```json
{
  "video_id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| video_id | number | 是 | 视频 ID |

### 响应

**200 OK**
```json
{
  "message": "like success"
}
```

**400 Bad Request**
```json
{
  "error": "video_id is required"
}
```
或
```json
{
  "error": "video not found"
}
```
或
```json
{
  "error": "user has liked this video"
}
```

---

### 2. 取消点赞

**POST** `/like/unlike`

**需要鉴权**：`Authorization: Bearer <token>`

### 请求体

```json
{
  "video_id": 1
}
```

### 响应

**200 OK**
```json
{
  "message": "unlike success"
}
```

**400 Bad Request**
```json
{
  "error": "user has not liked this video"
}
```
或
```json
{
  "error": "video not found"
}
```

---

### 3. 是否已点赞

**POST** `/like/isLiked`

**需要鉴权**：`Authorization: Bearer <token>`

未登录或 Token 无效时返回 `is_liked: false`。

### 请求体

```json
{
  "video_id": 1
}
```

### 响应

**200 OK**
```json
{
  "is_liked": true
}
```

---

### 4. 我点赞的视频列表

**POST** `/like/listMyLikedVideos`

**需要鉴权**：`Authorization: Bearer <token>`

### 响应

**200 OK**
```json
[
  {
    "id": 1,
    "author_id": 1,
    "username": "string",
    "title": "string",
    "description": "string",
    "play_url": "string",
    "cover_url": "string",
    "created_at": "2026-03-11T10:00:00Z",
    "likes_count": 0,
    "popularity": 0
  }
]
```

---

## 三、评论接口

### 1. 获取视频下所有评论（公开）

**POST** `/comment/listAll`

### 请求体

```json
{
  "video_id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| video_id | number | 是 | 视频 ID |

### 响应

**200 OK**
```json
[
  {
    "id": 1,
    "username": "string",
    "video_id": 1,
    "author_id": 1,
    "content": "string",
    "created_at": "2026-03-11T10:00:00Z"
  }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | number | 评论 ID |
| username | string | 评论者用户名 |
| video_id | number | 视频 ID |
| author_id | number | 评论者账户 ID |
| content | string | 评论内容 |
| created_at | string | 创建时间，ISO8601 |

**400 Bad Request**
```json
{
  "error": "video_id is required"
}
```
或
```json
{
  "error": "video not found"
}
```

---

### 2. 发布评论

**POST** `/comment/publish`

**需要鉴权**：`Authorization: Bearer <token>`

### 请求体

```json
{
  "video_id": 1,
  "content": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| video_id | number | 是 | 视频 ID |
| content | string | 是 | 评论内容 |

### 响应

**200 OK**
```json
{
  "message": "comment published successfully"
}
```

**400 Bad Request**
```json
{
  "error": "video_id is required"
}
```
或
```json
{
  "error": "content is required"
}
```
或
```json
{
  "error": "video not found"
}
```

---

### 3. 删除评论

**POST** `/comment/delete`

**需要鉴权**：`Authorization: Bearer <token>`

仅评论作者可删除自己的评论。

### 请求体

```json
{
  "comment_id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| comment_id | number | 是 | 评论 ID |

### 响应

**200 OK**
```json
{
  "message": "comment deleted successfully"
}
```

**400 Bad Request**
```json
{
  "error": "comment_id is required"
}
```
或
```json
{
  "error": "comment not found"
}
```
或
```json
{
  "error": "permission denied"
}
```

---

## 通用说明

- **Content-Type**：JSON 接口为 `application/json`；文件上传为 `multipart/form-data`
- **鉴权方式**：需鉴权接口在请求头添加 `Authorization: Bearer <token>`
- **错误响应**：统一为 `{"error": "错误描述"}`
- **401 Unauthorized**（鉴权失败）：`missing authorization header`、`invalid or expired token`、`token has been revoked`
- **静态资源**：使用本地存储时，上传文件可通过 `/static` 路径访问；使用七牛云 OSS 时，返回完整 URL
