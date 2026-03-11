# Account 服务 API 接口文档

**基础路径**：`/account`  
**端口**：8081（可通过 config 配置）

---

## 1. 注册

**POST** `/account/register`

### 请求体

```json
{
  "username": "string",
  "password": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

### 响应

**200 OK**
```json
{
  "message": "account created"
}
```

**400 Bad Request**
```json
{
  "error": "请求解析错误信息"
}
```

**409 Conflict**
```json
{
  "error": "username already exists"
}
```

---

## 2. 登录

**POST** `/account/login`

### 请求体

```json
{
  "username": "string",
  "password": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

### 响应

**200 OK**
```json
{
  "token": "string"
}
```

**401 Unauthorized**
```json
{
  "error": "invalid username or password"
}
```

---

## 3. 登出

**POST** `/account/logout`

**需要鉴权**：`Authorization: Bearer <token>`

### 响应

**200 OK**
```json
{
  "message": "account logged out"
}
```

**400 Bad Request**
```json
{
  "error": "accountID not found"
}
```

**401 Unauthorized**
```json
{
  "error": "missing authorization header"
}
```

---

## 4. 重命名

**POST** `/account/rename`

**需要鉴权**：`Authorization: Bearer <token>`

### 请求体

```json
{
  "new_username": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| new_username | string | 是 | 新用户名 |

### 响应

**200 OK**
```json
{
  "token": "string"
}
```

> 重命名成功后返回新 Token，需更新客户端存储的 token。

**400 Bad Request**
```json
{
  "error": "new_username is required"
}
```

**404 Not Found**
```json
{
  "error": "account not found"
}
```

**409 Conflict**
```json
{
  "error": "username already exists"
}
```

---

## 5. 修改密码

**POST** `/account/changePassword`

### 请求体

```json
{
  "username": "string",
  "old_password": "string",
  "new_password": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| old_password | string | 是 | 旧密码 |
| new_password | string | 是 | 新密码 |

### 响应

**200 OK**
```json
{
  "message": "successfully password changed"
}
```

**400 Bad Request**
```json
{
  "error": "unsuccessfully password changed"
}
```

---

## 6. 按 ID 查询

**POST** `/account/findByID`

### 请求体

```json
{
  "id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | number | 是 | 账户 ID |

### 响应

**200 OK**
```json
{
  "id": 1,
  "username": "string",
  "last_login_at": "2026-03-11T10:00:00Z",
  "last_logout_at": "2026-03-11T12:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | number | 账户 ID |
| username | string | 用户名 |
| last_login_at | string (可选) | 最近登录时间，ISO8601 |
| last_logout_at | string (可选) | 最近登出时间，ISO8601 |

**404 Not Found**
```json
{
  "error": "account not found"
}
```

---

## 7. 按用户名查询

**POST** `/account/findByUsername`

### 请求体

```json
{
  "username": "string"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |

### 响应

**200 OK**
```json
{
  "id": 1,
  "username": "string",
  "last_login_at": "2026-03-11T10:00:00Z",
  "last_logout_at": "2026-03-11T12:00:00Z"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| id | number | 账户 ID |
| username | string | 用户名 |
| last_login_at | string (可选) | 最近登录时间，ISO8601 |
| last_logout_at | string (可选) | 最近登出时间，ISO8601 |

**404 Not Found**
```json
{
  "error": "account not found"
}
```

---

## 8. 注销账户

**POST** `/account/cancel`

**需要鉴权**：`Authorization: Bearer <token>`

软删除当前账户，清除 Redis 中的 Token。注销后无法登录，用户名可被重新注册。

### 响应

**200 OK**
```json
{
  "message": "account cancelled"
}
```

**401 Unauthorized** - 同登出

---

## 9. Token 校验（内部接口）

**POST** `/internal/validate`

供 API Gateway 或其他服务校验 Token 并获取用户信息。

### 请求体

```json
{
  "token": "string"
}
```

> 也可通过请求头 `Authorization: Bearer <token>` 传递 token。

### 响应

**200 OK**
```json
{
  "account_id": 1,
  "username": "string"
}
```

**401 Unauthorized**
```json
{
  "error": "token required"
}
```

或
```json
{
  "error": "token has been revoked"
}
```

---

## 通用说明

- **Content-Type**：`application/json`
- **鉴权方式**：需鉴权接口在请求头添加 `Authorization: Bearer <token>`
- **错误响应**：统一为 `{"error": "错误描述"}`
