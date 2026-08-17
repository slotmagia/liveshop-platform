# Platform capability 目录

本目录承载 Platform 内部的跨业务基础设施边界。它们共享 Platform 进程、数据库、Admin contribution 和发布生命周期，不建立独立模块或服务。

| Capability | 责任 |
|---|---|
| `notification` | 短信/邮件 Provider、模板、通知事件、投递与 Provider 证据 |
| `storage` | 对象存储驱动、通道、密钥引用与健康状态 |
| `telemetry` | 全局埋点接收、去重、水位及广告归因投影 |
| `liveprovider` | 直播 Provider、商户/店铺分配、凭据引用与健康状态 |
| `localization` | 语言、翻译版本、发布工作流和机器翻译 Provider |

Identity 拥有验证码挑战/核验、游客风险、客服账号和商户治理；Catalog 拥有素材资产；Live 拥有房间、场次及直播专属指标。
