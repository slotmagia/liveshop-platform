// Package settings is the private HTTP wire contract of the admin surface
// settings capability.
package settings

import (
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type Document struct {
	Namespace string          `json:"namespace"`
	Value     json.RawMessage `json:"value"`
	Version   int64           `json:"version"`
	UpdatedBy string          `json:"updatedBy,omitempty"`
	UpdatedAt time.Time       `json:"updatedAt,omitempty"`
}

type CatalogFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type CatalogField struct {
	Key         string               `json:"key"`
	Label       string               `json:"label"`
	Type        string               `json:"type"`
	Help        string               `json:"help,omitempty"`
	Placeholder string               `json:"placeholder,omitempty"`
	Options     []CatalogFieldOption `json:"options,omitempty"`
}

type CatalogCategory struct {
	Key        string         `json:"key"`
	Label      string         `json:"label"`
	GroupKey   string         `json:"groupKey"`
	GroupLabel string         `json:"groupLabel"`
	Fields     []CatalogField `json:"fields"`
}

type CatalogGroup struct {
	Key        string            `json:"key"`
	Label      string            `json:"label"`
	Categories []CatalogCategory `json:"categories"`
}

type CatalogReq struct {
	g.Meta `path:"/settings/catalog" method:"get" tags:"Platform-配置" summary:"查询配置分类及表单元数据"`
}
type CatalogRes []CatalogGroup

type ListReq struct {
	g.Meta `path:"/settings" method:"get" tags:"Platform-配置" summary:"读取配置文档列表"`
}
type ListRes []Document

type GetReq struct {
	g.Meta    `path:"/settings/{namespace}" method:"get" tags:"Platform-配置" summary:"读取配置文档"`
	Namespace string `json:"namespace" in:"path"`
}
type GetRes Document

type PutReq struct {
	g.Meta          `path:"/settings/{namespace}" method:"put" tags:"Platform-配置" summary:"版本化写入配置文档"`
	Namespace       string          `json:"namespace" in:"path"`
	ExpectedVersion int64           `json:"expectedVersion"`
	Value           json.RawMessage `json:"value"`
}
type PutRes Document
