# Platform 后端局部规则

- HTTP 使用 GoFrame `ghttp` 和 `g.Meta`；controller 只做传输转换，不能访问数据库或具体 Store。
- gRPC controller 与 HTTP controller 调用同一个 application service；Proto 生成文件禁止手改。
- application/logic 不导入 gRPC、`ghttp`、数据库驱动或 controller。
- 业务领域包不得导入 GoFrame、gRPC、SQL、HTTP、transport 或 infrastructure。
- 所有 Store 和外部调用必须传递请求 `context.Context`；只允许进程启动和有界停机构造新的根 context。
- 状态写入必须检查预期旧状态或版本；可重试命令必须有稳定幂等键。
- 数据库与事件不能原子提交时使用 Outbox/Inbox，不得在事务提交后尽力发布关键事件。
- migration 只追加，不修改已发布 migration；权限、状态和数据迁移必须有明确收敛路径。
- 修改契约后运行 manifest compose check、Buf lint、Buf breaking 和生成漂移检查。
- 详细标准按需读取 `backend/docs/standards/`，业务事实读取 `backend/docs/domain/`。
