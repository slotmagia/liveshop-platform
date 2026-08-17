# API

该目录保存由各边界 `api` 契约生成或发布的 OpenAPI 等传输层产物。手写 Go 请求/响应类型仍位于 `internal/application/<surface>/api/http` 与 `internal/controlplane/<boundary>/api/http`，禁止在这里复制第二份 DTO。
