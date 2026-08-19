package i18n

import "github.com/gogf/gf/v2/frame/g"

type DriverField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Placeholder string `json:"placeholder,omitempty"`
	Help        string `json:"help,omitempty"`
}

type DriverDefinition struct {
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Fields      []DriverField `json:"fields"`
}

type DriversReq struct {
	g.Meta `path:"/i18n/drivers" method:"get" tags:"Platform-i18n" summary:"查询机器翻译驱动"`
}
type DriversRes struct {
	Items []DriverDefinition `json:"items"`
}

type ConfigView struct {
	Provider  string `json:"provider"`
	APIKeySet bool   `json:"apiKeySet"`
	Version   int64  `json:"version"`
}

type ConfigGetReq struct {
	g.Meta `path:"/i18n/config" method:"get" tags:"Platform-i18n" summary:"取机器翻译配置"`
}
type ConfigGetRes ConfigView

type ConfigUpdateReq struct {
	g.Meta          `path:"/i18n/config" method:"put" tags:"Platform-i18n" summary:"版本化保存机器翻译配置"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
	Provider        string `json:"provider"`
	APIKey          string `json:"apiKey"`
	APIKeyClear     bool   `json:"apiKeyClear"`
}
type ConfigUpdateRes ConfigView

type LocaleView struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type LocalesReq struct {
	g.Meta `path:"/i18n/locales" method:"get" tags:"Platform-i18n" summary:"平台目标语言目录"`
}
type LocalesRes struct {
	Items []LocaleView `json:"items"`
}

type EntityView struct {
	EntityType  string `json:"entityType"`
	Label       string `json:"label"`
	OwnerModule string `json:"ownerModule"`
	Field       string `json:"field"`
}

type EntitiesReq struct {
	g.Meta `path:"/i18n/entities" method:"get" tags:"Platform-i18n" summary:"可翻译内容类型"`
}
type EntitiesRes struct {
	Items   []EntityView `json:"items"`
	Locales []string     `json:"locales"`
}

type TextRow struct {
	EntityID   string `json:"entityId"`
	MerchantID int64  `json:"merchantId"`
	ShopID     int64  `json:"shopId"`
	Source     string `json:"source"`
	Value      string `json:"value"`
	Status     string `json:"status"`
	TextSource string `json:"textSource"`
	Stale      bool   `json:"stale"`
	Version    int64  `json:"version"`
}

type TextsListReq struct {
	g.Meta     `path:"/i18n/texts" method:"get" tags:"Platform-i18n" summary:"翻译工作台清单"`
	EntityType string `json:"entityType" in:"query" v:"required"`
	Locale     string `json:"locale" in:"query" v:"required"`
}
type TextsListRes struct {
	Items []TextRow `json:"items"`
}

type TextsPublishReq struct {
	g.Meta          `path:"/i18n/texts/publish" method:"post" tags:"Platform-i18n" summary:"发布译文"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
	EntityType      string `json:"entityType"`
	EntityID        string `json:"entityId"`
	Locale          string `json:"locale"`
	Value           string `json:"value"`
	MerchantID      int64  `json:"merchantId"`
	ShopID          int64  `json:"shopId"`
}
type TextsPublishRes struct {
	OK      bool  `json:"ok"`
	Version int64 `json:"version"`
}

type TextsFillReq struct {
	g.Meta     `path:"/i18n/texts/fill" method:"post" tags:"Platform-i18n" summary:"机器预填草稿"`
	CommandKey string `json:"commandKey"`
	EntityType string `json:"entityType"`
	Locale     string `json:"locale"`
}
type TextsFillRes struct {
	Provider string `json:"provider"`
	Filled   int    `json:"filled"`
	Skipped  int    `json:"skipped"`
}
