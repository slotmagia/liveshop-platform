# Shop 埋点事件调用规范

本文件是 Shop 浏览器上报 Platform 原始埋点的**独立调用契约**。线协议以 Manifest `platform.shop.track-events.create` 与 `g.Meta` 为准。领域事实见 [`../domain/事实.md`](../domain/事实.md)。

本仓库没有 `frontend-shop`。调用方是 Shop Host 装载的业务 contribution（Catalog 店铺页、Live 观看页等）。

## 1. 调用路径

```text
Shop 页面
  → @liveshops/host-sdk 的 context.api / iframeHttpClient
  → Gateway（按活动路由快照转发）
  → Platform POST /shop/platform/track-events
```

禁止：

- 直连 Platform `http://127.0.0.1:18082` 或容器服务名
- 手写 Gateway 地址、`Authorization`、Module Capability
- 从 URL、localStorage、cookie 读 token
- 调用旧路径 `POST /api/track/events`

| 项 | 值 |
|---|---|
| Operation | `platform.shop.track-events.create` |
| 方法 / 路径 | `POST /shop/platform/track-events` |
| 鉴权 | Identity Module Capability（`module-session`） |
| Surface | `shop`（请求头 `X-Liveshop-Surface: shop`） |
| 权限 | `platform.track-event.write` |
| 幂等 | `idempotent`；同一 `eventId` 重复写入计 `duplicates` |
| 探活 | `GET /shop/platform/health`（public，不校验 Capability） |

Host SDK 在 `api.request` 时自动设置 `Authorization: Bearer <moduleToken>`、`X-Liveshop-Surface`、`Content-Type: application/json`。页面只传绝对 path 和 JSON body。

## 2. 调用方前置条件

1. Identity 为当前 Shop contribution 签发 Module Capability，TTL ≤ 5 分钟，且同时携带 `merchantId > 0` 与 `shopId > 0`。
2. Capability 含权限 `platform.track-event.write`，且 `AllowsRequest` 覆盖 `POST /shop/platform/track-events`。
3. 调用方 contribution 的 `allowedRoutes` 必须声明该 method + prefix。空 `allowedRoutes` 只允许占位页。

```json
{
  "allowedRoutes": [
    {
      "methods": ["POST"],
      "prefix": "/shop/platform/track-events",
      "requiredPermissions": ["platform.track-event.write"]
    }
  ]
}
```

缺店铺上下文、缺权限或 route 不匹配：整单 **403**，Platform 不会从 body 补租户。

## 3. 租户与主体

| 字段 | 来源 | 客户端 |
|---|---|---|
| `merchantId` / `shopId` | Capability claims | **不要传**。若传必须与 claims 逐字相同，否则该行 `tenant_mismatch` |
| `surface` | 服务端固定写 `shop` | 不要传 |
| `subject` | Capability `subject` | 不要传 `uid` |
| `userAgent` / `ip` / `referer` | 服务端取请求 | 不要传 |
| `app` / `appId` / `commercialId` | 已废弃 | **禁止出现**，该行 `forbidden_field` |

店铺隔离只认 `merchant_id + shop_id`。`app_id` / `commercial_id` 不是运行时标识。

## 4. 请求

```http
POST /shop/platform/track-events
Content-Type: application/json
Authorization: Bearer <module-capability>
X-Liveshop-Surface: shop
```

可选请求头（整批共用，本切片只落到事件行，不写广告触点旁路表）：

| 头 | 说明 |
|---|---|
| `X-Ad-Touch-Id` | 已有触点数字 ID |
| `X-Ad-Touch-Type` | `gclid` / `fbclid` / `ttclid` / `msclkid` / `sccid` |

```json
{
  "events": [
    {
      "eventId": "8f1a2c3d-4e5f-6789-abcd-ef0123456789",
      "eventName": "page_view",
      "eventType": "page",
      "page": "/products/42",
      "component": "",
      "action": "",
      "bizType": "product",
      "bizId": "42",
      "sessionId": "sess-1",
      "anonymousId": "anon-1",
      "occurredAtMs": 1755600000000,
      "schemaVersion": 1,
      "props": {},
      "state": {},
      "extra": {},
      "liveContext": {}
    }
  ]
}
```

### 4.1 批次限制（整单失败）

| 条件 | HTTP | reason |
|---|---|---|
| `events` 为空 | 400 | `platform.telemetry.invalid` |
| 超过 100 条 | 400 | `platform.telemetry.invalid` |
| 正文超过 512 KiB | 400 | `platform.telemetry.invalid` |
| Capability 无完整店铺上下文 | 403 | `platform.telemetry.forbidden` |

### 4.2 单行字段

| 字段 | 必填 | 上限 | 说明 |
|---|---|---|---|
| `eventName` | 是 | 128 字节 | 稳定事件名，如 `page_view` |
| `eventType` | 是 | 32 字节 | 如 `page` / `action` / `session` |
| `eventId` | 否 | 64 字节 | 幂等键。重试必须保持不变。缺省由服务端生成后再去重 |
| `page` | 否 | 255 字节 | 页面路径 |
| `component` | 否 | 128 字节 | |
| `action` | 否 | 64 字节 | |
| `bizType` | 否 | 64 字节 | |
| `bizId` | 否 | 64 字节 | 字符串 |
| `sessionId` | 条件 | 96 字节 | 见 §5 |
| `anonymousId` | 否 | 96 字节 | 访客匿名 ID |
| `occurredAtMs` | 否 | — | 客户端毫秒时间。缺省用 `clientTs`，再缺省用服务器当前时间 |
| `clientTs` | 否 | — | 仅当未传 `occurredAtMs` 时作为发生时间 |
| `schemaVersion` | 否 | — | ≤0 视为 1 |
| `props` / `state` / `extra` / `liveContext` | 否 | 各 16 KiB JSON | 对象；`null` 按 `{}` |
| `merchantId` / `shopId` | 否 | — | 不推荐。出现则必须等于 claims |

时间窗口：发生时间须在 **过去 7 天至未来 10 分钟**。

去重键：`(merchantId, shopId, surface, eventId)`。`surface` 恒为 `shop`。

## 5. 核心事件附加规则

下列 `eventName` 除通用校验外还有条件字段。其它名字只做通用校验。

`session_enter` `session_ping` `session_exit` `page_enter` `page_meta` `page_exit` `page_view` `product_view` `product_card_exposure` `product_card_click` `add_to_cart` `payment_attempt` `payment_succeeded` `order_create` `ad_touch` `live_enter` `live_ping` `live_exit` `live_play_result`

| 事件 | 附加规则 |
|---|---|
| `session_*` | 必须有 `sessionId` |
| `page_enter` / `page_meta` / `page_exit` | 必须有 `sessionId`；`props.page_seq` 必须是正整数 |
| `page_view` | 不要求 `sessionId` / `page_seq` |
| `add_to_cart` | 若传 `props.sku_id`，必须是非负整数 |
| `payment_attempt` / `payment_succeeded` | 若传 `props.amount`，必须是非负整数 |
| 任意事件 | 若传 `liveContext.room_id`，必须是非负整数 |

本切片不校验直播房间是否存在，不写 `ad_touch` 旁路表，响应不含 `adTouch` 回执。

## 6. 响应

HTTP 200 只表示请求到达并完成批次处理。调用方必须读 `data`：

```json
{
  "code": 0,
  "data": {
    "accepted": 1,
    "duplicates": 0,
    "rejected": 0,
    "errors": []
  }
}
```

| 字段 | 含义 |
|---|---|
| `accepted` | 新插入行数 |
| `duplicates` | 已存在同一去重键，未覆盖 |
| `rejected` | 校验失败、未落库 |
| `errors[]` | 仅 rejected 行：`index`、`eventId`、`code`、`message` |

非法行不中止其余行。存储失败才整单回滚（HTTP 非 200）。

### 6.1 行级 `errors[].code`

| code | 含义 |
|---|---|
| `required` | 缺 `eventName` / `eventType`，或核心事件缺 `sessionId` / `page_seq` |
| `too_long` | 某字符串超上限 |
| `forbidden_field` | 出现 `app` / `appId` / `commercialId` |
| `uid_mismatch` | 出现 `uid` |
| `tenant_mismatch` | body 中的 `merchantId` / `shopId` 与 claims 不一致 |
| `invalid_time` | 发生时间超出窗口 |
| `json_too_large` | 某个 JSON 字段超过 16 KiB 或无法序列化 |
| `invalid_field` | 核心事件的 `props` / `liveContext` 数值不合法 |

## 7. Shop 页面调用示例

```ts
import type { RemoteModuleContext } from '@liveshops/host-sdk'

type TrackCreateRes = {
  accepted: number
  duplicates: number
  rejected: number
  errors?: Array<{ index: number; eventId?: string; code: string; message: string }>
}

export async function reportShopEvents(
  api: RemoteModuleContext['api'],
  events: Array<Record<string, unknown>>,
): Promise<TrackCreateRes> {
  return api.request<TrackCreateRes>('/shop/platform/track-events', {
    method: 'POST',
    body: JSON.stringify({ events }),
  })
}

// mount(container, context) 内：
await reportShopEvents(context.api, [{
  eventId: crypto.randomUUID(),
  eventName: 'page_view',
  eventType: 'page',
  page: '/home',
  sessionId: sessionStorage.getItem('ls-session-id') || undefined,
  anonymousId: localStorage.getItem('ls-anonymous-id') || undefined,
  occurredAtMs: Date.now(),
  props: {},
}])
```

重试同一事件必须复用同一个 `eventId`。不要为超时再生成新 ID，否则会写成第二条事实。

## 8. 总后台只读（对照，Shop 不要调）

| Operation | 方法 / 路径 | 权限 |
|---|---|---|
| `platform.admin.track-events.list` | `GET /admin/platform/track-events` | `platform.track-event.read` |

Admin 只浏览，不能改或删。Shop 页面不得调用 Admin 路径。

## 9. 本切片不做

- `ad_touch` 旁路建触点、返回触点回执
- 广告归因投影与 `/ad-attribution` 看板
- merch / live / admin 写入面（同一 usecase 可扩展，当前未暴露 HTTP）
- 游客无 Capability 的 `guest-session` 上报
