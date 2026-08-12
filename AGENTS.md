# LiveShop Platform 工程规则

- 四端技术 Host 和 Host Runtime 位于相邻的 `liveshop-gateway` 仓库；本仓库只包含 Platform 自有的模块贡献。
- 无状态数据面 Gateway 位于相邻的 `liveshop-gateway` 仓库；本仓库禁止包含 Gateway 实现。
- 业务模块位于独立仓库；本仓库禁止包含或导入领域实现和领域专属契约。
- 本仓库本身就是 Platform 控制面模块：Manifest 为根目录 `module.json`，Admin 贡献为 `frontend-admin`。后端根目录严格采用 `api/cmd/configs/contracts/deploy/docs/internal/migrations/pkg/tools`，Dockerfile 位于 `backend/deploy`；禁止恢复根级 `contracts/deploy/docs/tools` 或建立嵌套的 `modules` 目录。
- 后端代码位于 `backend/internal/platform`，顶层只允许 `app/application/cmd/common/registry`。HTTP 固定采用 GoFrame `ghttp`，构建链为 `api/v1(g.Meta) → controller → service interface → logic → router(g.Bind) → common/server`；公开 gRPC 契约位于 `backend/contracts/proto/platform/v1`，生成客户端位于 `backend/contracts/gen/go/platform/v1`，gRPC controller 复用同一 service interface 与 logic 实例，由 `common/grpcserver` 完成 mTLS、SPIFFE 授权、health 和服务注册。`app/bootstrap.go` 构造依赖，`registry.Dependencies` 传递完整进程依赖，`app/run.go` 只管理 HTTP/gRPC 生命周期；`common/server` 只创建 `ghttp.Server`、安装公共中间件并注册 router，禁止恢复 `net/http ServeMux` 或在其中编写业务 handler。
- Platform 进程配置只能来自 `-config` 指定的完整 YAML 文件；禁止通过环境变量、隐式覆盖或 fallback 修改监听地址、日志、数据库、密钥、身份、CORS、Cookie、gRPC TLS 等配置。
- 外部模块只能依赖已发布的 Platform contracts、Kernel 包、前端 SDK 和自身实现。
- 跨模块同步调用使用事实所有模块发布的公开契约；异步可靠流程使用事务 Outbox/Inbox。
- 浏览器中的 Platform contribution 使用与 contribution 绑定的 Module Session 经 Gateway 通信；Registry CI 和 Gateway 控制面调用使用短期工作负载身份，二者都不能绕过对应校验器。
- 所有可重试命令必须携带稳定的业务幂等键。
- 每项业务事实只能有一个事实源和一个所有模块/数据库。
- Host 禁止导入模块 `src`；只能加载 Registry 声明的不可变 iframe 或 remote ESM artifact。
- 浏览器认证和运行时流量只能通过 Gateway 的显式路由白名单到达 Platform；Registry 内部接口保持工作负载身份直连。
- 模块发布由 `(module_id, version, digest)` 唯一标识且不可变。

## 局部规则路由

- 修改后端前必须读取 `backend/AGENTS.md`；修改 Platform 领域实现还必须读取 `backend/internal/platform/AGENTS.md`。
- 修改前端 contribution 前必须读取 `frontend-admin/AGENTS.md`。
- 业务事实、不变量、状态转换、事务边界和外部契约分别以 `backend/docs/domain/` 下的五份文档为事实源。
- 普通业务模块不得复制 Platform 特有的 `registry` 组合方式；使用 `backend/tools/new-business-module.ps1` 生成独立业务模块骨架。
- 提交前根据变更范围运行 `verify-fast.ps1`、`verify-module.ps1` 或 `verify-release.ps1`，最低要求见 `backend/docs/standards/testing.md`。
