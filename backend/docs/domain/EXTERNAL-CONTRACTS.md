# Platform 外部契约

## HTTP

- 浏览器流量经 Gateway 显式白名单进入 Platform。
- Platform contribution 使用绑定 module/version/contribution/surface 的 Module Session。
- Registry CI 和 Gateway 控制面调用使用短期 workload identity。
- HTTP operation 的唯一机器契约由 `backend/contracts/manifest/platform/` fragments 组合为根 `module.json`。

## gRPC

- Proto：`backend/contracts/proto/platform/v1/platform.proto`。
- 传输：TLS 1.3 双向认证，证书 URI SAN 必须是授权 SPIFFE ID。
- Registry 快照 RPC 是只读、幂等调用，调用方必须设置 deadline。
- gRPC breaking change 由 Buf `FILE` 规则相对主分支或发布基线检查。

## Kernel

Platform 和 Gateway 只依赖已发布 Kernel 契约。当前 Realm claims 仍来自本地工作区中的未发布 Kernel 演进；独立发布前必须先发布包含该契约的新 Kernel 版本并移除本地 `replace`。
