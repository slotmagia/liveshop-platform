# Platform gRPC 契约

Platform 在 `server.grpc` 监听 `liveshop.platform.v1.PlatformRegistryService`。Proto 是唯一线协议事实源：

```text
backend/contracts/proto/platform/v1/platform.proto
backend/contracts/gen/go/platform/v1/
```

当前公开两个只读、安全且可重试的 RPC：

- `GetRouteSnapshot`：读取活动 HTTP 路由及其单调递增 revision；要求 `platform.registry.routes.read`。
- `GetCapabilityCatalog`：读取由不可变发布 Manifest 派生的能力目录；要求 `platform.registry.capabilities.read`。

两者复用 HTTP Registry API 的同一个 `service.Registry → logic → registry/module.Store` 实例，不建立第二份路由或能力事实源。

## 传输与身份

- 服务端强制 TLS 1.3、客户端证书和受信 CA 校验。
- 客户端证书必须带 URI SAN 形式的 SPIFFE ID。
- `workload_identity.gateway/release.spiffe_id` 固定 SPIFFE ID 到既有 subject 与权限集合；证书不能自行提升权限。
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
