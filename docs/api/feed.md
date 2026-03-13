# Feed 服务 API 接口文档

**基础路径**：`/feed`  
**端口**：8084（可通过 config 配置）

Feed 为聚合层，无独立 DB，依赖 Video、Social gRPC 获取数据。

---

## 一、最新流

### ListLatest 最新视频列表

**POST** `/feed/listLatest`

按发布时间倒序返回视频。

**需要鉴权**：`Authorization: Bearer <token>`

### 请求体

```json
{
  "limit": 20,
  "offset": 0
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | number | 否 | 每页数量，默认 20，最大 100 |
| offset | number | 否 | 偏移量，默认 0 |

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
    "popularity": 0,
    "is_liked": false
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
| is_liked | boolean | 当前用户是否已点赞 |

**401 Unauthorized**（鉴权失败）：`missing authorization header`、`invalid or expired token`、`token has been revoked`

---

## 二、关注流

### ListByFollowing 关注用户视频流

**POST** `/feed/listByFollowing`

**需要鉴权**：`Authorization: Bearer <token>`

返回当前用户关注的人所发布的视频，按时间倒序。无关注时返回空列表。

### 请求体

```json
{
  "limit": 20,
  "offset": 0
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | number | 否 | 每页数量，默认 20，最大 100 |
| offset | number | 否 | 偏移量，默认 0 |

### 响应

**200 OK**
```json
[
  {
    "id": 2,
    "author_id": 5,
    "username": "followed_user",
    "title": "string",
    "description": "string",
    "play_url": "string",
    "cover_url": "string",
    "created_at": "2026-03-11T10:00:00Z",
    "likes_count": 10,
    "popularity": 100,
    "is_liked": true
  }
]
```

**401 Unauthorized**（鉴权失败）：`missing authorization header`、`invalid or expired token`、`token has been revoked`

---

## 三、热度流

### ListByPopularity 热度排序视频列表

**POST** `/feed/listByPopularity`

**需要鉴权**：`Authorization: Bearer <token>`

按热度（popularity）倒序返回视频。

### 请求体

```json
{
  "limit": 20,
  "offset": 0
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| limit | number | 否 | 每页数量，默认 20，最大 100 |
| offset | number | 否 | 偏移量，默认 0 |

### 响应

**200 OK**

格式同 ListLatest，按 `popularity` 降序排列。

---

## 通用说明

- **Content-Type**：`application/json`
- **鉴权方式**：所有接口均需 `Authorization: Bearer <token>`
- **错误响应**：`{"error": "错误描述"}`
- **401 Unauthorized**：`missing authorization header`、`invalid or expired token`、`token has been revoked`
