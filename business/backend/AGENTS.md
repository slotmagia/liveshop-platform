# Platform Backend 工程规则

- 后端实现 Registry、Navigation、Settings、Audit、Notification、Storage、Telemetry/Attribution、Live Provider、Localization 和受信工作负载快照。
- Provider Secret 只保存密文或密钥引用，读取默认脱敏；通知、埋点和 Provider 回调必须幂等、去重并保留 UNKNOWN 恢复路径。
- 禁止保存 Identity 验证码核验结果、游客风险决策、客服账号、商户治理状态、Catalog 素材资产或 Live 房间/场次事实。
- HTTP Admin 路由必须先验证 Identity Module Capability，再分别校验 Manifest 权限与 method/path；请求字段不能覆盖 capability 中的身份或作用域。
- `GetRouteSnapshot` 仅授权 Gateway workload；`GetActiveCapabilitySnapshot` 仅授权 Identity workload。gRPC 同时校验 TLS client certificate、SPIFFE、JWT subject 与 method permission。
- 激活事务内完成 release 状态、revision、活动权限目录和审计写入。任何校验失败都不得改变 Active、Revision、权限目录或审计。
- 活动贡献的 `(surface, groupId)` 标题与排序一致性必须在事务写入前验证，冲突返回领域错误并 fail closed。
- Platform 只能构造 `modulesession.Verifier` 验证冻结的 Module Capability 线协议；禁止构造 issuer、读取签名私钥或调用 Identity 目录计算用户授权。
- 旧授权表是迁移输入，不是运行时事实源。导出命令可重复、digest 稳定；finalize 必须验证 Identity 持久导入回执。
- 代码修改后运行 `gofmt`、`go test ./...`，涉及 MySQL 时运行对应 integration test。
