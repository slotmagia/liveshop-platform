# liveshop-platform 工程规则

分层开发总册：本仓库根 [`docs/开发规范.md`](../../docs/开发规范.md)。本仓库入口：[`backend/docs/文档目录.md`](./backend/docs/文档目录.md)。Platform HTTP operation id 必须含 surface，见通用规范第 6 章。

## 唯一职责

- Platform 是模块控制面和跨业务基础设施控制面，拥有 Registry、Navigation、Settings、Audit、Notification、Storage、Telemetry/Attribution、Live Provider 与 Localization。
- Identity 是浏览器用户身份、组织、角色、策略、Subject grant、DataScope、有效授权和 Module Capability 签发的唯一事实源。
- Platform 不得实现登录、用户会话、IAM 写入、有效权限计算、权益投影或 Module Capability 私钥签发。
- Platform notification 只拥有短信/邮件 Provider、可复用模板库、通知事件策略和投递证据，不拥有 Identity 验证码挑战或核验结果。
- Platform storage 只拥有对象存储驱动、通道、密钥引用和健康状态；素材资产属于 Catalog。Platform live-provider 只拥有 Provider、分配和凭据引用；直播房间与场次属于 Live。
- Platform telemetry/attribution 只拥有全局埋点和可重建归因投影；游客风险决策、客服账号和商户能力治理属于 Identity。
- 经营分析、店铺访问和广告商品映射属于 Catalog；Platform 只向 Catalog 发布版本化 telemetry 事件或公开投影，不保存 Catalog 经营指标副本。
- `application/admin` 只承载 Platform 控制面页面；不得新增 `application/merch` 或 `frontend-merch`。
- `controlplane/provisioning` 只服务受信工作负载。浏览器不得直连该边界。

## 安全边界

- Admin contribution 只接受 Identity 签发、TTL 不超过五分钟、绑定 module/release/contribution/surface/route/revision 的 Module Capability。
- Platform 只保存 Identity issuer 与公钥并执行 fail-closed 验签；禁止保存 Module Capability 私钥或提供签发接口。
- Gateway 只能读取路由快照；Identity 使用独立 mTLS/SPIFFE 工作负载读取活动能力快照。两类权限不得互换。
- Registry 激活是权限定义与贡献目录的唯一发布点。活动能力快照必须携带 registry revision，且只包含当前活动发布。
- 同一 `(surface, groupId)` 的活动导航贡献必须具有完全相同的 `groupTitle/groupSort`；冲突激活必须整体回滚。

## 目录与发布

- `module.json` 由兄弟仓 `liveshop-protocol/platform/manifest/platform` 组合生成，禁止手工制造第二份契约。
- 改 Manifest 碎片、`g.Meta`、contribution 或前端 API 路径后，先 compose，再按仓库根 [`docs/命名规范检查.md`](../docs/命名规范检查.md) 做命名检查。
- Protocol 变更必须同步生成 Go 代码、Manifest、测试和 baseline。
- 浏览器页面只调用 Manifest 声明的 Gateway 路由，不直接调用内部 gRPC 或工作负载接口。
- 旧授权数据只能通过 `authorizationexport` 两阶段迁移：导出并由 Identity 持久导入后，携带匹配 digest 的 Identity receipt 才能 finalize 删除旧表。不得保留双运行或 fallback。

## 验证

- 后端：`go test ./...`
- 前端：`npm run build`
- 协议：`buf lint`、生成代码、Manifest compose/check
- 部署：`docker compose -f backend/deploy/compose.local.yml config --quiet`
