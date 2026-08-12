# 后端目录标准

普通业务模块采用有限上下文纵向目录，不复制 Platform 特有 Registry：

```text
backend/
├─ api/<surface>/<module>/v1
├─ contracts/proto/<module>/v1
├─ internal/<module>/
│  ├─ app
│  ├─ domain
│  ├─ application
│  ├─ infrastructure/postgres
│  └─ transport
│     ├─ http
│     └─ grpc
└─ migrations
```

依赖方向固定为 `transport → application → domain`，`infrastructure → domain/application ports`。组合根负责注入；domain 不知道 GoFrame、gRPC 或 PostgreSQL。

Platform 现有 `api → controller → service → logic` 是控制面参考实现。新业务模块使用 `new-business-module.ps1` 创建，不通过复制 Platform 目录起步。
