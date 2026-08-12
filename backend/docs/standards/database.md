# 数据库标准

- 每个业务事实只有一个数据库所有者；禁止跨模块直连数据表。
- migration 只追加，带稳定编号，生产由独立 migration job 在应用启动前执行。
- 同一数据库必须一致的写入放入同一事务；状态更新校验旧状态或版本。
- 数据库调用传递请求 context，并设置 statement/transaction deadline。
- 连接池必须显式配置 max open、max idle、max lifetime 和 max idle time。
- 热路径表采用规范化模型和必要索引；禁止用单行无限增长 JSON 文档承载整个业务域。
- 测试必须覆盖真实 PostgreSQL 的约束、隔离级别、并发冲突、提交失败和重试收敛。
