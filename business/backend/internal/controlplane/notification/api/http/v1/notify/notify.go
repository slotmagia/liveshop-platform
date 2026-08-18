package notify

import "github.com/gogf/gf/v2/frame/g"

type Recipients struct {
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
}

type DispatchReq struct {
	g.Meta      `path:"/deliveries" method:"post" tags:"Platform-通知投递" summary:"统一 Dispatch 通知"`
	EventKey    string            `json:"eventKey"`
	DeliveryKey string            `json:"deliveryKey"`
	MerchantID  int64             `json:"merchantId"`
	ShopID      int64             `json:"shopId"`
	Recipients  Recipients        `json:"recipients"`
	Variables   map[string]string `json:"variables"`
	NotBefore   string            `json:"notBefore"`
	Locale      string            `json:"locale"`
}

type DeliveryResult struct {
	DeliveryID string `json:"deliveryId"`
	Channel    string `json:"channel"`
	Status     string `json:"status"`
	Deduped    bool   `json:"deduped"`
}

type DispatchRes struct {
	Deliveries []DeliveryResult `json:"deliveries"`
}

type GetReq struct {
	g.Meta     `path:"/deliveries/{deliveryId}" method:"get" tags:"Platform-通知投递" summary:"读取一次渠道投递"`
	DeliveryID string `json:"deliveryId" in:"path"`
}

type GetRes struct {
	DeliveryID   string `json:"deliveryId"`
	DeliveryKey  string `json:"deliveryKey"`
	EventKey     string `json:"eventKey"`
	Channel      string `json:"channel"`
	Status       string `json:"status"`
	MerchantID   int64  `json:"merchantId"`
	ShopID       int64  `json:"shopId"`
	LastError    string `json:"lastError,omitempty"`
	AttemptCount int    `json:"attemptCount"`
}
