package notifyevents

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type ChannelPolicy struct {
	Enabled      bool   `json:"enabled"`
	TemplateCode string `json:"templateCode,omitempty"`
}

type Event struct {
	EventKey        string                   `json:"eventKey"`
	ModuleID        string                   `json:"moduleId"`
	ModuleName      string                   `json:"moduleName"`
	OperationID     string                   `json:"operationId"`
	Title           string                   `json:"title"`
	Variables       []string                 `json:"variables"`
	AllowedChannels []string                 `json:"allowedChannels"`
	DefaultDispatch string                   `json:"defaultDispatch"`
	DispatchMode    string                   `json:"dispatchMode"`
	DelaySeconds    int                      `json:"delaySeconds"`
	Channels        map[string]ChannelPolicy `json:"channels"`
	PolicyVersion   int64                    `json:"policyVersion"`
	UpdatedAt       time.Time                `json:"updatedAt"`
}

type ListReq struct {
	g.Meta  `path:"/notify-events" method:"get" tags:"Platform-通知事件" summary:"查询通知事件目录"`
	Module  string `json:"module" in:"query"`
	Channel string `json:"channel" in:"query"`
	Keyword string `json:"keyword" in:"query"`
}
type ListRes []Event

type GetReq struct {
	g.Meta   `path:"/notify-events/{eventKey}" method:"get" tags:"Platform-通知事件" summary:"读取一个通知事件"`
	EventKey string `json:"eventKey" in:"path"`
}
type GetRes Event

type ReplacePolicyReq struct {
	g.Meta          `path:"/notify-events/{eventKey}/policy" method:"put" tags:"Platform-通知事件" summary:"版本化保存渠道策略"`
	EventKey        string                   `json:"eventKey" in:"path"`
	CommandKey      string                   `json:"commandKey"`
	ExpectedVersion int64                    `json:"expectedVersion"`
	DispatchMode    string                   `json:"dispatchMode"`
	DelaySeconds    int                      `json:"delaySeconds"`
	Channels        map[string]ChannelPolicy `json:"channels"`
}
type ReplacePolicyRes Event

type Delivery struct {
	DeliveryID   string    `json:"deliveryId"`
	DeliveryKey  string    `json:"deliveryKey"`
	Channel      string    `json:"channel"`
	Status       string    `json:"status"`
	Recipient    string    `json:"recipient"`
	LastError    string    `json:"lastError,omitempty"`
	AttemptCount int       `json:"attemptCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ListDeliveriesReq struct {
	g.Meta   `path:"/notify-events/{eventKey}/deliveries" method:"get" tags:"Platform-通知事件" summary:"查询事件投递记录"`
	EventKey string `json:"eventKey" in:"path"`
}
type ListDeliveriesRes []Delivery
