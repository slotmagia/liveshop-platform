# LiveShop Platform 框架

本仓库是域中立的 LiveShop 控制面和 Platform 系统模块，负责身份、IAM、平台配置、审计、模块注册、公共契约以及 Platform 自有的前端贡献。浏览器 Gateway/Host 骨架和业务模块分别在独立仓库中开发与发布。

## 工作区边界

| 边界 | 职责 |
|---|---|
| 相邻的 `../liveshop-gateway` 仓库 | 浏览器入口、四端 Host、Host Runtime、无状态 HTTP/WebSocket Gateway |
| 根目录 `module.json`、`backend`、`frontend-admin` | 后台账号与会话、IAM/组织事实、不可变发布、平台配置、审计及 Platform Admin 贡献 |
| 外部业务模块仓库 | 领域后端、自有数据库、自有 gRPC 契约和可选前端贡献 |
| `backend/contracts` | 域中立 Manifest 契约与 Schema |
| [`github.com/lvtuopen-ai/kernel-go`](https://github.com/lvtuopen-ai/kernel-go) | 已发布的域中立 Go 运行时 |
| `packages/*` | 供 Gateway 和模块使用的 Host SDK、Design Tokens 与公共 UI 契约 |

Host 不包含 Catalog、Order、Payment、Live 或 Merchant 业务实现。独立发布的模块通过自己的 `module.json` 向 Host 提供 page、slot、widget 或 action。

Platform 模块与独立业务模块采用相同的根级结构：

```text
liveshop-platform/
├─ module.json
├─ backend/
│  ├─ api/                  # OpenAPI 等生成/发布产物
│  ├─ cmd/                  # 独立运维命令
│  ├─ configs/              # 完整 YAML 配置模板与本地配置
│  ├─ contracts/            # Manifest、Platform Proto 与生成客户端独立 Go module
│  ├─ deploy/               # 镜像与本地 Compose
│  ├─ docs/                 # 架构、协同与运行文档
│  ├─ migrations/           # 只追加数据库迁移
│  ├─ pkg/                  # 稳定公开 Go 包
│  ├─ tools/                # 校验、启动与冒烟脚本
│  └─ internal/platform/
│     ├─ app/              # bootstrap 与运行生命周期
│     ├─ application/platform/
│     │  ├─ api/v1/        # g.Meta 请求/响应契约
│     │  ├─ controller/    # 薄 HTTP/gRPC 适配器
│     │  ├─ service/       # 应用边界接口
│     │  ├─ logic/         # 用例编排与错误映射
│     │  └─ router/        # ghttp 分组、鉴权与 g.Bind
│     ├─ common/           # 中间件、请求上下文、响应和服务组合根
│     ├─ cmd/
│     └─ registry/         # 进程依赖集合及 audit/iam/identity/module/settings 事实子包
├─ frontend-admin/
└─ packages/
```

仓库中不存在嵌套的 `modules` 工作区。Platform Admin 页面使用 Host SDK，通过 Gateway 和与 contribution 绑定的 Module Session 通信。Gateway 路由快照读取和模块发布注册分别使用短期工作负载身份调用 Platform 内部接口。

Admin 与 Merchant 是同一后台外壳中的两个独立授权 surface。Admin 认证 `PLATFORM` 身份，Merchant 认证 `MERCHANT` 身份。二者可以保持相同布局，但任何 realm 都不能申请另一个 surface 的 contribution 或 Module Session。

## 当前运行组件

- Platform：HTTP `8082`、gRPC `9082`（TLS 1.3 mTLS + SPIFFE）
- 动态 Gateway：HTTP `8081`
- Admin Host：`5173`
- Merchant Host：`5174`
- Shop Host：`5175`
- Live Host：`5176`
- Platform IAM Admin artifact：`5180`

相邻的 `liveshop-gateway` 仓库提供可运行的数据面 Gateway，`liveshop-modular` 是 Catalog 示例模块。本地脚本和 Compose 将这些同级仓库集成运行，但不会把源码所有权重新移入 Platform。

文档入口：

- [`backend/docs/TEAM-DEVELOPMENT-STANDARD.md`](./backend/docs/TEAM-DEVELOPMENT-STANDARD.md)：跨仓多人协作，以及详细的后端/前端工程目录、分层、命名、测试、文档生产和通信规则。
- [`backend/docs/ARCHITECTURE.md`](./backend/docs/ARCHITECTURE.md)：运行模型和仓库边界。
- [`backend/docs/PLATFORM-CONTROL-PLANE.md`](./backend/docs/PLATFORM-CONTROL-PLANE.md)：Platform/模块所有权和控制面不变量。
- [`backend/docs/MODULE-DEVELOPMENT.md`](./backend/docs/MODULE-DEVELOPMENT.md)：模块 Manifest、Host 和通信接入契约。
- [`backend/docs/AGENT-BUSINESS-DEVELOPMENT.md`](./backend/docs/AGENT-BUSINESS-DEVELOPMENT.md)：让编码 Agent/工程师按架构开发业务的输入、边界与交付闭环；不定义产品运行时 Agent。

## 创建业务模块

编码 Agent 或工程师应从模板创建独立业务仓库，不复制 Platform 控制面实现：

```powershell
./backend/tools/new-business-module.ps1 `
  -Destination ../liveshop-catalog `
  -ModuleId catalog `
  -ModuleName "Catalog" `
  -GoModule github.com/liveshop/catalog
```

模板生成仓库级/后端级 `AGENTS.md`、领域事实与一致性文档、GoFrame HTTP 纵向切片、gRPC Proto 入口、migration、YAML 配置、Manifest 和验证脚本。先补全业务事实与不变量，再实现用例。

## 验证

```powershell
./backend/tools/verify-fast.ps1       # 本地快速反馈
./backend/tools/verify-module.ps1     # race、覆盖率、可选 PostgreSQL 集成
./backend/tools/verify-release.ps1    # 漏洞、契约兼容、镜像
```

`verify.ps1` 是发布门禁别名。PostgreSQL 集成测试通过 `PLATFORM_TEST_DATABASE_URL` 显式启用；CI 默认提供隔离数据库。

## 本地开发

```powershell
npm install
Push-Location ../liveshop-gateway
npm install
Pop-Location
./backend/tools/launch-local.ps1
```

`launch-local.ps1` 会选择第一组完整可用的端口，生成 `.run/platform.yaml`，通过 `-config` 启动 Platform 服务和前端 Host，等待所有依赖就绪，并写入 `.run/dev-profile.json`。Platform 不接受环境变量配置；外部模块仓库可以在注册时读取运行配置。

使用标准端口手动启动：

```powershell
./backend/tools/start-dev.ps1
../liveshop-modular/tools/launch-local.ps1
```

运行 Platform 冒烟并停止本地进程：

```powershell
./backend/tools/smoke.ps1
./backend/tools/smoke-platform-controls.ps1
./backend/tools/stop-dev.ps1
```

`start-dev.ps1` 会在启动前检查所有请求端口。端口冲突时可覆盖脚本参数，具体参数见 `Get-Help ./backend/tools/start-dev.ps1 -Detailed` 或脚本参数块。

运行时的 Identity、Registry、IAM、Settings 和 Audit 使用事务型 PostgreSQL。启动 Platform 前必须按顺序应用 `backend/migrations` 下的 SQL；生产 migration 不会创建租户用户或超级管理员。本地 Compose 会另外执行 `backend/tools/seed-iam-local.sql`，Admin 使用 `admin / admin`，Merchant 使用 `merch@sufeipay.com / 123456`。这些弱凭据仅限本地开发；生产部署必须显式创建初始管理员，禁止复用本地凭据。
