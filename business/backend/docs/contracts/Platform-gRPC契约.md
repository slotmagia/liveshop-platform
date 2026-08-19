# Platform gRPC 契约

Platform 在 `server.grpc` 监听 `liveshop.platform.v1.PlatformRegistryService`。Proto 是唯一线协议事实源：

```text
liveshop-protocol/platform/proto/platform/v1/platform.proto
liveshop-protocol/platform/gen/go/platform/v1/
```

当前公开 `PlatformRegistryService` 的两个只读、安全且可重试的 RPC：

- `GetRouteSnapshot`：读取活动 HTTP 路由及其单调递增 revision；要求 `platform.registry.routes.read`。
- `GetActiveCapabilitySnapshot`：读取由不可变发布 Manifest 派生的能力目录；要求 `platform.registry.active-capabilities.read`。

两者属于 `controlplane/provisioning` 边界，与该边界的 HTTP 接口复用同一个 `service.Provisioning → logic → biz.Release → data/mysql` 实例，不建立第二份路由或能力事实源。

同一 gRPC 端口另挂 `liveshop.platform.v1.PlatformNotificationService`（`controlplane/notification`），不并入 Registry 服务：

- `Dispatch`：业务事务提交后按 `eventKey` 投递；要求 `platform.notify-event.dispatch`。
- `GetDelivery`：读取一次渠道投递证据；同一权限。

gRPC 与 HTTP 各自拥有请求/响应契约和错误映射：状态码由领域错误直接推导，不从 HTTP 状态码转换。服务名与「方法 → 工作负载权限」映射由各边界门面声明，`common/grpcserver` 只负责安装。

## 传输与身份

- 服务端强制 TLS 1.3、客户端证书和受信 CA 校验。
- 客户端证书必须带 URI SAN 形式的 SPIFFE ID。
- `workload_identity.grpc.gateway/identity.spiffe_id` 固定 SPIFFE ID 到互斥的最小权限集合；证书不能自行提升权限。Release CI 只使用独立 HTTP bearer 配置，不进入 gRPC 信任表。
- 标准 `grpc.health.v1.Health` 仅向已经通过 mTLS 且 SPIFFE ID 受信的工作负载开放。
- 调用方必须使用 Manifest 中的 `recommendedDeadlineMs` 设置 deadline，并只对标记为 `safe` 或 `idempotent` 的调用重试。

Platform 不读取环境变量。部署必须通过 `-config` 指定完整 YAML 配置，并将证书 Secret 以只读文件挂载到下列配置项声明的路径：

```text
grpc.tls.certificate_file
grpc.tls.private_key_file
grpc.tls.client_ca_file
```

本地开发由 `backend/cmd/grpccerts` 生成最长 24 小时有效的临时证书；生产禁止使用该本地 CA，必须接入部署环境的工作负载 CA/SPIFFE 证书交付机制。

修改 Proto 后运行：

```powershell
./backend/tools/generate-proto.ps1
```
