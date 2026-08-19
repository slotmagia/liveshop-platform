# telemetry

负责跨业务埋点事件接收、`(merchant_id, shop_id, surface, event_id)` 去重、处理水位、可重建投影和广告归因。游客风险决策由 Identity risk 形成，本 capability 不直接封禁主体或推进业务终态。

当前切片只落原始事件写入与 Admin 只读浏览。广告触点旁路表、投影作业和归因看板不在本切片。店铺隔离只用 Identity Module Capability 的 `merchant_id + shop_id`，不接受 `app_id` / `commercial_id`。
