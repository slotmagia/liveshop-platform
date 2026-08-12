// Package settings 定义 Platform Settings HTTP 契约。
package settings

import (
	"encoding/json"

	"github.com/gogf/gf/v2/frame/g"
	platformsettings "github.com/liveshop-platform/module-platform/internal/platform/registry/settings"
)

type ListReq struct {
	g.Meta `path:"/settings" method:"get" tags:"Platform-配置" summary:"读取配置文档列表"`
}
type ListRes []platformsettings.Document

type GetReq struct {
	g.Meta    `path:"/settings/{namespace}" method:"get" tags:"Platform-配置" summary:"读取配置文档"`
	Namespace string `json:"namespace" in:"path"`
}
type GetRes platformsettings.Document

type PutReq struct {
	g.Meta          `path:"/settings/{namespace}" method:"put" tags:"Platform-配置" summary:"版本化写入配置文档"`
	Namespace       string          `json:"namespace" in:"path"`
	ExpectedVersion int64           `json:"expectedVersion"`
	Value           json.RawMessage `json:"value"`
}
type PutRes platformsettings.Document
