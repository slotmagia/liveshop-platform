package trackevents

import (
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type Event struct {
	MerchantID    int64           `json:"merchantId"`
	ShopID        int64           `json:"shopId"`
	Surface       string          `json:"surface"`
	EventID       string          `json:"eventId"`
	EventType     string          `json:"eventType"`
	EventName     string          `json:"eventName"`
	Page          string          `json:"page"`
	Component     string          `json:"component"`
	Action        string          `json:"action"`
	BizType       string          `json:"bizType"`
	BizID         string          `json:"bizId"`
	SessionID     string          `json:"sessionId"`
	AnonymousID   string          `json:"anonymousId"`
	Subject       string          `json:"subject"`
	ClientTs      int64           `json:"clientTs"`
	OccurredAt    time.Time       `json:"occurredAt"`
	ReceivedAt    time.Time       `json:"receivedAt"`
	SchemaVersion int             `json:"schemaVersion"`
	LiveContext   json.RawMessage `json:"liveContext"`
	Props         json.RawMessage `json:"props"`
	State         json.RawMessage `json:"state"`
	Extra         json.RawMessage `json:"extra"`
	UserAgent     string          `json:"userAgent"`
	IP            string          `json:"ip"`
	Referer       string          `json:"referer"`
	AdTouchID     int64           `json:"adTouchId"`
	ClickIDType   string          `json:"clickIdType"`
	CreatedAt     time.Time       `json:"createdAt"`
}

type ListReq struct {
	g.Meta      `path:"/track-events" method:"get" tags:"Platform-上报事件" summary:"查询上报事件"`
	MerchantID  int64  `json:"merchantId" in:"query"`
	ShopID      int64  `json:"shopId" in:"query"`
	Surface     string `json:"surface" in:"query"`
	EventName   string `json:"eventName" in:"query"`
	EventType   string `json:"eventType" in:"query"`
	Subject     string `json:"subject" in:"query"`
	AnonymousID string `json:"anonymousId" in:"query"`
	StartMs     int64  `json:"startMs" in:"query"`
	EndMs       int64  `json:"endMs" in:"query"`
	Page        int    `json:"page" in:"query"`
	PageSize    int    `json:"pageSize" in:"query"`
}

type ListRes struct {
	Items []Event `json:"items"`
	Total int     `json:"total"`
}
