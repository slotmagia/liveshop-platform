# notification

负责通知事件目录、渠道策略、可复用模板库、Dispatch、Delivery/Attempt/Inbox 和 Provider 证据。

短信/邮件 **通道、驱动、密钥** 分别由 `sms`、`email` capability 拥有；站内信 inbox 单例在本 capability。本 capability 决定 **发什么、发哪几个渠道、用哪份模板、同步还是异步**，发送时调用短信/邮件 capability。Identity auth 只提交稳定 delivery key，不向本 capability 泄露验证码核验状态。

实施说明：仓库工作区 [`docs/legacy-system-inventory/module-splitting/11-Platform通知事件落地.md`](../../../../../../../docs/legacy-system-inventory/module-splitting/11-Platform通知事件落地.md)。
