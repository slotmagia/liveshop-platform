// Package audit is the private HTTP wire contract of the admin surface audit
// capability.
package audit

import (
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type Event struct {
	ID           string          `json:"id"`
	OccurredAt   time.Time       `json:"occurredAt"`
	ActorSubject string          `json:"actorSubject"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resourceType"`
	ResourceID   string          `json:"resourceId"`
	Result       string          `json:"result"`
	Details      json.RawMessage `json:"details"`
}

type ListReq struct {
	g.Meta `path:"/audit/events" method:"get" tags:"Platform-审计" summary:"读取审计事件"`
	Limit  int `json:"limit" in:"query"`
}
type ListRes []Event
