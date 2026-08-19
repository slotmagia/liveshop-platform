package i18n

import (
	"github.com/gogf/gf/v2/frame/g"
)

type TextsListReq struct {
	g.Meta     `path:"/texts" method:"get" tags:"Platform-i18n-internal" summary:"已发布译文组"`
	EntityType string `json:"entityType" in:"query" v:"required"`
	Locale     string `json:"locale" in:"query" v:"required"`
	MerchantID int64  `json:"merchantId" in:"query"`
	ShopID     int64  `json:"shopId" in:"query"`
}

type PublishedText struct {
	EntityID string `json:"entityId"`
	Value    string `json:"value"`
	Version  int64  `json:"version"`
}

type TextsListRes struct {
	Items []PublishedText `json:"items"`
}

type LocalesReq struct {
	g.Meta `path:"/locales" method:"get" tags:"Platform-i18n-internal" summary:"平台目标语言"`
}

type LocaleView struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type LocalesRes struct {
	Items []LocaleView `json:"items"`
}

type EntitiesReq struct {
	g.Meta `path:"/entities" method:"get" tags:"Platform-i18n-internal" summary:"可翻译类型"`
}

type EntityView struct {
	EntityType  string `json:"entityType"`
	Label       string `json:"label"`
	OwnerModule string `json:"ownerModule"`
	Field       string `json:"field"`
}

type EntitiesRes struct {
	Items []EntityView `json:"items"`
}

type SourcesUpdateReq struct {
	g.Meta        `path:"/sources" method:"put" tags:"Platform-i18n-internal" summary:"投影源文快照"`
	EntityType    string `json:"entityType"`
	EntityID      string `json:"entityId"`
	MerchantID    int64  `json:"merchantId"`
	ShopID        int64  `json:"shopId"`
	Source        string `json:"source"`
	SourceVersion int64  `json:"sourceVersion"`
}

type SourcesUpdateRes struct {
	OK bool `json:"ok"`
}
