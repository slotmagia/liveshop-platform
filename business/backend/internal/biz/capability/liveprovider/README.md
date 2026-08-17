# liveprovider

负责直播 Provider、商户/店铺分配、凭据引用和健康状态。Live 消费版本化 Provider 分配，不保存 Provider Secret。

当前已实现 Admin Provider 目录：`Provider(code)` 是当前配置，`ProviderVersion(code, version)` 是不可变快照；密钥使用配置的 AES-256-GCM key 加密，关联数据绑定 `code`（历史密文仍可用原 AAD 解密），读接口只返回是否配置、掩码和 key ID。物理删除不属于状态机，退役进入 `RETIRED` 终态。

商户/店铺分配仍属于本 capability 的后续纵向切片，不得重新放回 Live。分配必须引用已存在的 Provider 版本；Live 只通过内部版本化快照契约消费。
