# Platform 事务边界

| 用例 | 同一事务内必须完成 |
|---|---|
| 模块激活 | 锁定 Registry、校验活动路由、更新活动版本、同步权限状态、递增 revision、写审计 |
| 模块停用 | 锁定 Registry、移除活动版本、退役权限、递增 revision、写审计 |
| IAM 写入 | 条件更新实体、更新关联、递增租户 IAM revision、写审计 |
| 账号修改 | 条件更新账号、必要时撤销 session family、写审计 |
| Settings 写入 | expectedVersion 条件更新、递增版本、写审计 |
| Refresh 轮换 | 锁定 token/session、旧 token 标记 USED、插入新 token、更新 session |

所有事务使用调用方 context 并设置有界超时。可重试范围只包括 PostgreSQL serialization failure 和 deadlock；等待必须响应 context cancellation。

当前 Platform 没有跨数据库关键写入或消息发布。未来增加异步事实时，必须在权威写入事务中追加 Outbox，由可重放 relay 发布，并由消费者 Inbox 去重。
