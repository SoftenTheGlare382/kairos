# IM 服务 API 接口文档

**基础路径**：`/im`  
**端口**：8085（可通过 config 配置）

**私聊规则**：仅互相关注用户可无障碍聊天；非互关时，每人只能向对方发送一条「介绍」消息，需互关后才能继续。

---

## 一、REST 接口（需鉴权）

### 1. 发送消息

**POST** `/im/send`

#### 请求体

```json
{
  "receiver_id": 2,
  "content": "你好"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| receiver_id | number | 是 | 接收方用户 ID |
| content | string | 是 | 消息内容 |

#### 响应

**200 OK** 返回消息对象
```json
{
  "id": 1,
  "conversation_id": 1,
  "sender_id": 1,
  "receiver_id": 2,
  "content": "你好",
  "created_at": "2026-03-14T10:00:00Z"
}
```

**403 Forbidden** 非互关且已发过一条
```json
{
  "error": "you have already sent your one intro message; mutual follow required for more"
}
```

---

### 2. 模糊搜索消息（依赖 Meilisearch）

**POST** `/im/search`

#### 请求体

```json
{
  "query": "关键词",
  "limit": 20,
  "offset": 0
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| query | string | 是 | 搜索关键词 |
| limit | number | 否 | 每页条数，默认 20 |
| offset | number | 否 | 偏移量，默认 0 |

#### 响应

**200 OK**
```json
{
  "messages": [
    {
      "id": 1,
      "conversation_id": 1,
      "sender_id": 1,
      "receiver_id": 2,
      "content": "包含关键词的消息内容",
      "created_at": "2026-03-14T10:00:00Z"
    }
  ],
  "total": 5
}
```

未配置 Meilisearch（`MEILISEARCH_HOST` 为空）时返回空列表。

---

### 3. 会话列表

**POST** `/im/conversations`

#### 请求体

```json
{
  "limit": 20,
  "offset": 0
}
```

#### 响应

**200 OK**
```json
[
  {
    "id": 1,
    "user_a": 1,
    "user_b": 2,
    "last_message_at": "2026-03-14T10:00:00Z",
    "created_at": "2026-03-14T09:00:00Z",
    "unread_count": 3
  }
]
```

| 字段 | 说明 |
|------|------|
| unread_count | 当前用户在该会话中的未读消息数 |

---

### 4. 标记已读

**POST** `/im/read`

点进会话或手动标记已读后，该会话的未读数清零。`ListMessages` 也会在拉取消息时自动标记已读。

#### 请求体

```json
{
  "conversation_id": 1
}
```

#### 响应

**200 OK** `{"message":"ok"}`

---

### 5. 消息历史

**POST** `/im/messages`（拉取时自动标记该会话已读）

#### 请求体

```json
{
  "conversation_id": 1,
  "limit": 20,
  "offset": 0
}
```

#### 响应

**200 OK**
```json
[
  {
    "id": 1,
    "conversation_id": 1,
    "sender_id": 1,
    "receiver_id": 2,
    "content": "你好",
    "created_at": "2026-03-14T10:00:00Z"
  }
]
```

---

## 二、WebSocket（实时消息）

**GET** `/im/ws`

- 建立连接后，**10 秒内**必须发送首条鉴权消息（JSON）：
  ```json
  {"type":"auth","token":"<jwt>"}
  ```
- 服务端校验通过后返回 `{"type":"auth_ok"}`，失败返回 `{"type":"auth_err","error":"..."}`
- 鉴权成功后，接收方在线时会实时收到新消息（JSON 格式，同消息对象）
- **不在 URL query 中传 token**，避免日志、Referer 等泄露
