# LiveShop Platform Control Plane

`liveshop-platform` 是模块控制面，负责模块发布注册、活动版本、路由与活动能力快照、导航目录、平台设置和审计。它不再是浏览器用户 IAM 或授权事实源。

## 边界

| 事实/能力 | 所有者 |
|---|---|
| 模块 release、活动版本、registry revision | Platform Registry |
| 路由、权限定义、贡献与前端契约目录 | Platform Registry 的活动发布 |
| 用户、组织、角色、策略、DataScope、有效授权 | Identity |
| Module Capability 签发与授权过滤后的浏览器 runtime | Identity |
| Gateway 路由代理与 Host | Gateway |

Platform Admin 页面接受 Identity 签发的 Module Capability。Platform 仅用配置的 Identity issuer/public key 做验签，不保存签发私钥，也不计算用户权限。

## 目录

```text
business/
├─ module.json                 # 由 protocol/manifest 组合生成
├─ backend/
│  ├─ internal/application/admin
│  ├─ internal/controlplane/provisioning
│  ├─ internal/biz
│  ├─ internal/data
│  ├─ migrations
│  └─ docs
├─ frontend-admin/             # Registry / Settings / Audit contribution
└─ packages/                   # Host SDK 与设计令牌
protocol/
├─ proto/platform/v1
└─ manifest/platform
```

`provisioning` 是内部工作负载边界：Gateway 读取路由快照，Identity 读取带 registry revision 的完整活动能力快照。浏览器不能直连这些接口。

## 旧授权数据交接

历史授权表不再被运行时代码读取。使用 `cmd/authorizationexport` 生成确定性导出，Identity 持久导入并出具匹配 digest/rowCount 的 receipt 后，才能执行 `-finalize` 删除旧表。详见 [授权交接.md](backend/docs/授权交接.md)。

## 验证

```powershell
go -C backend test ./...
npm run build
docker compose -f backend/deploy/compose.local.yml config --quiet
```
