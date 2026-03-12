# Social 服务 API 接口文档

**基础路径**：`/social`  
**端口**：8083（可通过 config 配置）  
**鉴权**：所有接口需 JWT Token（`Authorization: Bearer <token>`）

---

## 一、关注 / 取关

### 1. 关注

**POST** `/social/follow`

### 请求体

```json
{
  "following_id": 2
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| following_id | number | 是 | 被关注者的 account_id |

### 响应

**200 OK**
```json
{
  "message": "follow success"
}
```

**400 Bad Request**
```json
{
  "error": "already following"
}
```
或 `user not found`、`cannot follow yourself` 等。

---

### 2. 取关

**POST** `/social/unfollow`

### 请求体

```json
{
  "following_id": 2
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| following_id | number | 是 | 被取关者的 account_id |

### 响应

**200 OK**
```json
{
  "message": "unfollow success"
}
```

**400 Bad Request**
```json
{
  "error": "not following"
}
```

---

## 二、粉丝 / 关注列表（分页）

### 3. 粉丝列表

**POST** `/social/followers`

### 请求体

```json
{
  "user_id": 1,
  "page": 1,
  "page_size": 20
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | number | 否 | 查询目标用户 ID；不传则查当前登录用户 |
| page | number | 否 | 页码，默认 1 |
| page_size | number | 否 | 每页条数，默认 20，最大 100 |

### 响应

**200 OK**
```json
{
  "list": [
    {"id": 2, "username": "alice"},
    {"id": 3, "username": "bob"}
  ],
  "total": 100,
  "has_more": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| list | array | 用户列表 |
| total | number | 总条数 |
| has_more | boolean | 是否有下一页 |

---

### 4. 关注列表

**POST** `/social/following`

### 请求体

```json
{
  "user_id": 1,
  "page": 1,
  "page_size": 20
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | number | 否 | 查询目标用户 ID；不传则查当前登录用户 |
| page | number | 否 | 页码，默认 1 |
| page_size | number | 否 | 每页条数，默认 20，最大 100 |

### 响应

**200 OK**
```json
{
  "list": [
    {"id": 4, "username": "charlie"}
  ],
  "total": 50,
  "has_more": false
}
```

---

## 三、gRPC 接口（供 Feed 服务调用）

### GetFollowingIDs

获取某用户关注的用户 ID 列表。

**请求**：`GetFollowingIDsRequest{ follower_id: uint32 }`  
**响应**：`GetFollowingIDsResponse{ following_ids: []uint32 }`

---

## 四、数据库

Social 服务使用 `kairos_db`，表 `follows`：

- follower_id：关注者
- following_id：被关注者
- 唯一约束：(follower_id, following_id)
