// Package audit 定义 Platform Audit HTTP 契约。
package audit

import (
	"github.com/gogf/gf/v2/frame/g"
	platformaudit "github.com/liveshop-platform/module-platform/internal/platform/registry/audit"
)

type ListReq struct {
	g.Meta `path:"/audit/events" method:"get" tags:"Platform-审计" summary:"读取审计事件"`
	Limit  int `json:"limit" in:"query"`
}
type ListRes []platformaudit.Event
