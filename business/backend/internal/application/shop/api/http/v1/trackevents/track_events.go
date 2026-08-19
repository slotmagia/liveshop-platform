package trackevents

import "github.com/gogf/gf/v2/frame/g"

type EventInput struct {
	EventID       string         `json:"eventId"`
	EventName     string         `json:"eventName"`
	EventType     string         `json:"eventType"`
	Page          string         `json:"page"`
	Component     string         `json:"component"`
	Action        string         `json:"action"`
	BizType       string         `json:"bizType"`
	BizID         string         `json:"bizId"`
	SessionID     string         `json:"sessionId"`
	AnonymousID   string         `json:"anonymousId"`
	OccurredAtMs  int64          `json:"occurredAtMs"`
	ClientTs      int64          `json:"clientTs"`
	SchemaVersion int            `json:"schemaVersion"`
	LiveContext   map[string]any `json:"liveContext"`
	Props         map[string]any `json:"props"`
	State         map[string]any `json:"state"`
	Extra         map[string]any `json:"extra"`
	MerchantID    int64          `json:"merchantId"`
	ShopID        int64          `json:"shopId"`
	App           string         `json:"app"`
	AppID         int64          `json:"appId"`
	CommercialID  int64          `json:"commercialId"`
	UID           int64          `json:"uid"`
}

type CreateReq struct {
	g.Meta `path:"/track-events" method:"post" tags:"Platform-上报事件" summary:"上报店铺埋点事件"`
	Events []EventInput `json:"events"`
}

type ItemError struct {
	Index   int    `json:"index"`
	EventID string `json:"eventId,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateRes struct {
	Accepted   int         `json:"accepted"`
	Duplicates int         `json:"duplicates"`
	Rejected   int         `json:"rejected"`
	Errors     []ItemError `json:"errors,omitempty"`
}
