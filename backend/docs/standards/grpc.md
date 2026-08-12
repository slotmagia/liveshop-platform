# gRPC 标准

- Proto package 使用 `liveshop.<module>.v1`，生成文件禁止手改。
- controller 只转换 Proto 与 application model，HTTP/gRPC 复用同一 application service。
- 服务端强制 TLS 1.3 mTLS，并按 SPIFFE ID 和 full method 授权。
- 客户端必须设置现实 deadline；只有声明为幂等且使用稳定业务键的调用才能配置自动重试。
- 参数、未找到、冲突、前置条件和内部错误分别映射到标准 gRPC codes。
- 注册标准 Health 服务；drain 前先切换为 `NOT_SERVING`，然后有界优雅停止。
- CI 执行 Buf lint、生成漂移和相对发布基线的 breaking check。
