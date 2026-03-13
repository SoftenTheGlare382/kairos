# Worker 服务说明

Worker 为异步消费者，消费 RabbitMQ 中的 4 个队列，处理点赞、评论、关注、热度相关事件。

## 依赖

- **RabbitMQ**：必选，队列由 Video/Social 服务发布
- **Redis**：必选，用于更新热榜 ZSET
- **MySQL**：启动时全量同步用，与 Video 服务共用数据库

## 队列与处理逻辑

| 队列 | 发布方 | 处理逻辑 |
|------|--------|----------|
| like.events | Video（点赞/取消点赞） | 更新 Redis `feed:hot:likes` ZSET（video_id → 点赞数） |
| comment.events | Video（发布/删除评论） | 仅 ack（可扩展） |
| social.events | Social（关注/取关） | 仅 ack（可扩展） |
| video.popularity.events | Video（点赞/评论/收藏/播放） | 更新 Redis `feed:hot` ZSET（video_id → 热度） |

## Redis 键约定

| 键 | 类型 | 说明 |
|----|------|------|
| feed:hot | ZSET | 视频热度排行，score=加权和（点赞20%+评论40%+收藏30%+观看10%） |
| feed:hot:likes | ZSET | 点赞数排行，member=video_id，score=likes_count |

Feed 服务当前使用 Video gRPC `ListByPopularity` 从 MySQL 获取热度流；后续可改为从 Redis ZSET 读取以提升性能。

## 启动时全量同步

Worker 启动时会从 MySQL `videos` 表全量同步 `likes_count`、`popularity` 到 Redis ZSET，再启动 4 个消费者。这样新部署或 Redis 清空后，热榜数据能正确初始化。同步失败仅打日志，不阻塞启动。

## 启动

```bash
# 需先启动 RabbitMQ、Redis、MySQL
cd services/worker && go run ./cmd
```

或在项目根目录执行 `./start.sh`，会一并启动 Worker。

## 配置

通过 `config.yaml` 或 `config.env` 配置：

- `RABBITMQ_URL`：RabbitMQ 连接地址，默认 `amqp://guest:guest@127.0.0.1:5672/`
- `REDIS_HOST`、`REDIS_PORT` 等：Redis 连接

## 事件格式（JSON）

### like.events
```json
{"video_id":1,"account_id":2,"delta":1}
```
- delta: 1=点赞，-1=取消

### comment.events
```json
{"video_id":1,"delta":1}
```
- delta: 1=发布，-1=删除

### social.events
```json
{"follower_id":1,"following_id":2,"action":"follow"}
```
- action: "follow" | "unfollow"

### video.popularity.events
```json
{"video_id":1,"delta":1}
```
