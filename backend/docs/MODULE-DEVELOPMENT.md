# LiveShop 模块开发与接入规范

> 状态：当前生效
> 适用范围：所有独立业务模块及 Platform 自有 contribution
> 所有者：Platform Architecture / Platform Engineering
> 最后核对：2026-08-12

本文定义模块接入 Platform、Gateway 和四端 Host 的机器契约及实施步骤。跨团队分工、工程结构、文档生产、一致性和交付门禁以 [`TEAM-DEVELOPMENT-STANDARD.md`](./TEAM-DEVELOPMENT-STANDARD.md) 为准；本文只保留模块接入所需的协议细节，避免在业务模块仓库复制通用规则。

## 1. 独立模块仓库

推荐由编码 Agent/工程师运行 Platform 提供的脚手架创建；这里的 Agent 是研发执行者，不是待嵌入业务系统的 Agent Runtime：

```powershell
./backend/tools/new-business-module.ps1 -Destination ../liveshop-inventory -ModuleId inventory -ModuleName "Inventory" -GoModule github.com/liveshop/inventory
```

```text
<module-repository>/
├─ AGENTS.md
├─ README.md
├─ module.json
├─ ownership.yaml
├─ dependency-policy.yaml
├─ backend/
│  ├─ tools/
│  ├─ deploy/Dockerfile
│  ├─ configs/<module>.yaml
│  ├─ pkg/
│  ├─ cmd/
│  ├─ migrations/
│  ├─ api/
│  ├─ contracts/             # 模块拥有的公开 gRPC Proto 和生成客户端
│  ├─ docs/
│  └─ internal/<module>/
├─ frontend-admin/            # 可选 iframe
├─ frontend-merch/            # 可选 iframe
├─ frontend-shop/             # 可选 remote ESM
└─ frontend-live/             # 可选 remote ESM
```

只创建模块实际贡献的 surface。每个 Go 后端和 npm artifact 必须能够独立测试与构建。业务模块只能依赖发布版本的 Platform contracts、`kernel-go`、Host SDK、Design Tokens 和允许的公共 UI；本地相邻 checkout 只用于开发替换，发布构建不能依赖其他仓库源码路径。

`host-runtime` 属于 Gateway 实现，不是业务模块依赖。业务模块不得导入其他模块的 `internal`、`src`、DAO 或数据库。

## 2. 创建模块前必须确认

开始编码前形成一页模块定义：

| 项目 | 必须回答的问题 |
|---|---|
| 模块身份 | module ID、显示名、初始 SemVer 和负责人是什么？ |
| 事实源 | 哪些聚合、表和字段由本模块最终负责？租户复合键是什么？ |
| 不变量 | 正常、失败、重复和并发时绝不能破坏什么？ |
| 状态机 | 有哪些状态、合法/非法转换、终态和副作用？ |
| Surface | Admin、Merch、Shop、Live 中实际需要哪些？ |
| 同步协议 | 哪些 HTTP 面向浏览器，哪些 gRPC 面向服务？ |
| 异步协议 | 有哪些事件；Outbox/Inbox、顺序和恢复如何保证？ |
| 外部副作用 | 支付、库存、通知或第三方调用如何幂等、查询和恢复？ |
| 交付 | 迁移、镜像、健康检查、告警、验证和发布由谁负责？ |

事实源、关键不变量或状态机无法确定时停止实现，向业务/架构负责人提出最小问题。

## 3. Module Manifest

清单固定为 `<module-repository>/module.json`，结构由 Platform 发布的 `backend/contracts/modulemanifest` 和 `backend/contracts/schemas/module-manifest.schema.json` 约束。

### 3.1 发布身份

- `apiVersion` 固定为 `liveshop.io/v1`。
- `kind` 固定为 `ModuleRelease`。
- `metadata.id` 使用小写 kebab-case，首次发布后不能重命名。
- `metadata.version` 使用三段 SemVer。
- `(module_id, version, digest)` 唯一且不可变；内容变化必须发布新版本。
- 所有前端 artifact 的版本必须等于模块版本。
- 生产完整性必须为真实 `sha256:<hex>`；`sha256:dev-*` 只允许本地开发。

### 3.2 HTTP route 与 operation

`spec.backend.httpRoutes[]` 声明 surface、模块命名空间前缀和公开 operation：

```json
{
  "surface": "merch",
  "prefix": "/merch/inventory",
  "operations": [
    {
      "id": "inventory.merch.items.list",
      "method": "GET",
      "path": "/merch/inventory/items",
      "summary": "查询库存",
      "description": "返回当前租户和数据范围内的库存。",
      "authentication": "module-session",
      "idempotency": "safe",
      "requiredPermissions": ["inventory.item.read"],
      "requestFields": [],
      "responses": [{"status": 200, "description": "库存列表", "fields": []}]
    }
  ]
}
```

规则：

- surface 只能是 `admin`、`merch`、`shop`、`live` 或仅服务端使用的 `internal`。
- 浏览器前缀必须包含 surface 和模块命名空间。
- 活动发布中的 `surface + prefix` 不能冲突；不能利用重叠前缀制造隐式优先级。
- `backend.origin` 是 Gateway 访问模块后端的 origin，不是前端 artifact 地址。
- operation 必须声明稳定 ID、完整 method/path、鉴权方式、幂等语义、权限、请求字段及响应状态。
- 鉴权方式使用 `module-session`、`workload-identity` 或经安全评审允许的 `public`。
- 幂等语义使用 `safe`、`idempotent` 或 `non-idempotent`；可重试写接口还必须定义稳定幂等键。

### 3.3 权限与 contribution

`spec.permissions` 只声明模块能力，不能向用户或角色授权。权限码使用 `<resource>.<action>`，所有 contribution、route 和 action 引用的权限必须在同一 Manifest 中定义。

贡献类型为 `page`、`slot`、`widget` 或 `action`：

- `page` 声明绝对 `route`；其他类型声明 Host 已支持的 `outlet`。
- contribution ID 在模块版本内唯一，推荐 `<module>.<surface>.<capability>`。
- `requiredPermissions` 控制 contribution 可见性。
- `allowedRoutes` 以 method、path prefix 和权限声明 Module Session 的最小调用范围，并且必须是对应 surface route 的子集。
- iframe 声明 `entry`、`integrity`；remote ESM 还要声明 `exportName`。
- `frontend.component` 是实际导出组件；props、events、actions 必须给出字段和权限。
- action `target` 只能引用稳定 operation ID、Host event、导航 route 或 module export，不能引用源码路径或内部函数。

Platform IAM 是授权事实源。Manifest 只定义能力目录；Host 的菜单/插槽过滤和后端授权均不能信任 Manifest 自行授予用户权限。

## 4. Host outlet 与前端协议

当前稳定 outlet：

- Admin：`admin.dashboard.widgets`
- Merchant：`merch.dashboard.widgets`
- Shop：`shop.home.hero`、`shop.product.grid`、`shop.checkout.payment-methods`
- Live：`live.player.overlay`、`live.room.product-panel`、`live.room.interaction-panel`

模块消费 Host 已发布 outlet，不能自行创造 Host 布局契约。

当前 Host 协议版本为 `HOST_PROTOCOL = 1`。HostContext 包含 module/version/contribution、surface、Gateway 地址、locale、theme、permissions、tenant 和五分钟有效的 contribution-bound Module Session；数据范围由签名 claims 提供，前端不能扩大。

### 4.1 iframe

1. iframe 调用 Host SDK `connectToHost()`。
2. iframe 向 `document.referrer` 的精确 origin 发送 `LIVESHOP_MODULE_READY`。
3. Host 校验 `event.source` 和 artifact origin。
4. Host 返回 `LIVESHOP_HOST_CONTEXT`。
5. iframe 校验 source、origin 和 protocol。

禁止 `postMessage(..., "*")`，禁止从 URL 传 token，禁止读取父窗口私有 DOM 或 Store。Admin contribution 还必须使用 Platform 发布的 `@liveshop/admin-ui` 与 `admin.css`，不能复制一套后台基础组件样式。

### 4.2 remote ESM

- Host 获取源码并校验 SHA-256 后动态 import。
- 导出名必须等于 Manifest `exportName`。
- `mount(container, context)` 只操作传入容器。
- 全局事件、订阅、计时器和资源必须由 `unmount` 清理。
- API 只通过 `context.api`，导航只通过 `context.navigate`。

## 5. 浏览器 HTTP

模块前端使用 Host SDK 客户端，经 Gateway 携带：

```http
Authorization: Bearer <module-session>
X-Liveshop-Surface: admin|merch|shop|live
Content-Type: application/json
```

模块 router 必须再次校验签名、issuer、audience、subject、jti、有效期、module ID、surface、header、allowed method/path、permissions 和 data scopes。`appId`、`merchantId`、`departmentIds`、owner subject 只能来自已验证 claims，不能由 body、query 或自定义 header 覆盖。

GoFrame 约定：

- Req/Res 使用 `XxxReq`、`XxxRes`；method/path 写入 `g.Meta`。
- controller 只调用本 surface service。
- router 挂 Module Session、统一 ResponseHandler，再绑定 controller。
- 写接口使用明确 method，禁止 `ALL`。
- 成功信封为 `{"code":0,"data":...}`；失败使用正确非 2xx 状态和 `{"code":<stable-code>,"message":"..."}`。
- Gateway 是浏览器 CORS 最终边界，会移除模块上游 CORS header。

## 6. 服务间 gRPC

契约位置：

```text
backend/contracts/proto/<module>/v1/<module>.proto
backend/contracts/gen/go/<module>/v1/
```

- package 使用 `liveshop.<module>.v1`。
- 字段号发布后不得复用；兼容增加字段，不兼容变更发布新 package 版本。
- 生成文件禁止手工编辑。
- Manifest 的 service、contractVersion、endpoint 和 transportSecurity 必须与实现一致。
- 服务端只做 Proto 与 `biz` 转换；HTTP 和 gRPC 使用同一领域应用实例。
- 参数错误用 `InvalidArgument`，不存在用 `NotFound`，并发冲突用 `Aborted`/`FailedPrecondition`，未知错误用 `Internal`。
- 生产通信强制 TLS 1.3 双向认证和授权的 SPIFFE 工作负载身份。
- tenant/业务 ID 显式进入请求；调用方设置 deadline，只有幂等 RPC 可自动重试。
- 服务注册 gRPC health，并在优雅停机前切换为 `NOT_SERVING`。

## 7. 控制面、发现与能力目录

| 操作 | 方法与路径 | 身份/权限 |
|---|---|---|
| 注册发布 | `POST /internal/v1/module-registry/releases` | CI workload；`registry.release.write` |
| 激活版本 | `POST /internal/v1/module-registry/activate` | CI workload；`registry.activation.write` |
| Gateway 获取路由 | `GET /internal/v1/module-registry/routes` | Gateway workload；`registry.routes.read` |
| 服务/Agent 获取能力 | `GET /internal/v1/module-registry/capabilities` | workload；`registry.capabilities.read` |
| Host 获取贡献 | `GET /runtime/v1/contributions?surface=...` | 用户访问身份，IAM 过滤 |
| Host 申请会话 | `POST /runtime/v1/module-sessions` | 用户访问身份 |
| Gateway 管理界面能力目录 | `GET /runtime/v1/module-catalog` | Platform 用户；`platform.registry.manage` |
| Platform 模块页能力目录 | `GET /admin/platform/registry/capabilities` | Platform Admin Module Session |

能力目录的唯一事实源是 Registry 中由 `(module_id, version, digest)` 标识的不可变 Manifest。路由快照、Host contribution、权限目录和能力目录必须由同一活动发布派生；Gateway 管理界面只展示，不保存、探测或覆盖能力。

注册/激活必须保持：

1. 同一模块版本只有一个摘要和能力契约；
2. 激活版本已经成功注册；
3. 注册失败不留下部分能力；
4. 激活失败不改变当前版本、权限目录和 revision；
5. 重复注册相同内容、重复激活当前版本收敛且不递增 revision。

消费方服务必须先读取能力目录再调用；目录没有声明的 operation 视为未公开。浏览器调用使用 Gateway + Module Session，服务调用使用公开 gRPC + 工作负载身份，写调用遵守声明的幂等与并发前置条件。这是业务能力发现协议，不要求引入 Agent Run、Task、Lease、Checkpoint 等运行时模型。

## 8. 发布生命周期

1. CI 验证后端、契约和所有声明的前端 artifact。
2. 构建不可变后端镜像和前端 artifact。
3. 生成真实摘要并冻结 `module.json`。
4. 使用短期工作负载身份注册发布，但不假设立即生效。
5. 对未激活版本运行集成测试。
6. 激活指定版本；Platform 在串行化事务中校验冲突并递增 route revision。
7. Gateway 拉取新快照；刷新失败保留最后有效快照。
8. Host 按 surface 重新获取 contribution。

禁止覆盖同版本清单、直接修改活动 Registry 数据或用隐藏路由“热修复”。

## 9. 新模块实施顺序

1. 使用 `new-business-module.ps1` 建立 `AGENTS.md`、ownership、依赖策略、领域文档和独立构建单元。
2. 起草 Manifest、权限、HTTP operations、Proto 和事件 schema。
3. 实现 `biz/model` 的事实、不变量和状态机。
4. 实现 `biz` 用例和端口、`data` 事务适配器及 migrations。
5. 写流程同时实现幂等命令、条件更新和所需 Outbox/Inbox；生产运行时缺少数据库/消息依赖时 fail-fast，不能回退内存。
6. 以 `<surface>/<capability>` 完成 `api → transport/http → application → domain port ← infrastructure` 纵向链路；transport 不直接访问 SQL。
7. 实现 `transport/grpc` 到同一 application service 的适配，HTTP 与 gRPC 不各自维护业务逻辑。
8. 实现 contribution；API 路径只放在 `views/<domain>/api`，入口只负责 Host 握手和 mount/unmount。
9. 更新 `go.work`、npm workspaces、CI、Compose、端口、Secret、健康检查、注册和安全停止脚本。
10. 完成 Manifest、路由、鉴权、tenant 隔离、数据库、事件、artifact 和真实 Gateway 链路验证。

开发启动器至少应：构建后端、启动 HTTP/gRPC 和 artifact、保存 PID/日志、等待依赖健康、注册本地 origin/entry、申请真实 Module Session，并通过 Gateway 请求一个模块接口。

## 10. UTF-8

源码、HTML、CSS、JavaScript、JSON、SQL、配置和文档统一 UTF-8；`.editorconfig` 是编辑器事实源。HTML 在任何非 ASCII 内容前声明 `<meta charset="UTF-8" />`，容器前端使用共享 Nginx 配置为 HTML/CSS/JS/JSON/SVG 声明 UTF-8。

乱码按源码字节 → 构建产物 → HTTP `charset` → 镜像重建 → 浏览器缓存的顺序排查，不直接修改 `dist` 或替换本来正确的中文源码。

## 11. 接入完成条件

- Manifest decode、重复 contribution、route 冲突和 artifact 版本/摘要校验通过。
- 每个声明 surface 的真实路由、错误 token/surface/path/permission 和 tenant 隔离测试通过。
- HTTP 信封、gRPC 状态映射、health、deadline 和工作负载授权经过验证。
- 每个 artifact 可独立构建并导出正确入口，Host 无业务硬编码。
- 注册 → 激活 → contribution → Module Session → Gateway → 模块链路通过。
- 写功能完成全部适用的一致性故障测试，并断言最终状态和副作用次数。
- 模块 README、AGENTS、业务一致性文档、迁移、运行手册和消费者同步完成。
