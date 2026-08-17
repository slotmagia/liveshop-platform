// Package sms defines the Admin HTTP wire contract for Platform SMS.
// Secret values are write-only.
package sms

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type CredentialChange struct {
	Mode  string `json:"mode"`
	Value string `json:"value,omitempty"`
}

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
	g.Meta `path:"/sms/drivers" method:"get" tags:"Platform-短信" summary:"查询短信驱动及表单元数据"`
}
type DriversRes []DriverDefinition

type Channel struct {
	ID              int64             `json:"id"`
	Code            string            `json:"code"`
	Name            string            `json:"name"`
	Driver          string            `json:"driver"`
	Region          string            `json:"region"`
	Priority        int               `json:"priority"`
	Enabled         bool              `json:"enabled"`
	Lifecycle       string            `json:"lifecycle"`
	PublicConfig    map[string]string `json:"publicConfig"`
	SecretMasks     map[string]string `json:"secretMasks"`
	CredentialKeyID string            `json:"credentialKeyId,omitempty"`
	Version         int64             `json:"version"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type ListChannelsReq struct {
	g.Meta    `path:"/sms/channels" method:"get" tags:"Platform-短信" summary:"查询短信通道目录"`
	Keyword   string `json:"keyword" in:"query"`
	Driver    string `json:"driver" in:"query"`
	Lifecycle string `json:"lifecycle" in:"query"`
}
type ListChannelsRes []Channel

type PutChannelReq struct {
	g.Meta          `path:"/sms/channels/{code}" method:"put" tags:"Platform-短信" summary:"版本化新增或修改短信通道"`
	Code            string                      `json:"code" in:"path"`
	CommandKey      string                      `json:"commandKey"`
	ExpectedVersion int64                       `json:"expectedVersion"`
	Name            string                      `json:"name"`
	Driver          string                      `json:"driver"`
	Region          string                      `json:"region"`
	Priority        int                         `json:"priority"`
	PublicConfig    map[string]string           `json:"publicConfig"`
	Secrets         map[string]CredentialChange `json:"secrets"`
}
type PutChannelRes Channel

type EnableChannelReq struct {
	g.Meta          `path:"/sms/channels/{code}/enable" method:"post" tags:"Platform-短信" summary:"启用短信通道"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type EnableChannelRes Channel
type DisableChannelReq struct {
	g.Meta          `path:"/sms/channels/{code}/disable" method:"post" tags:"Platform-短信" summary:"停用短信通道"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type DisableChannelRes Channel

type RetireChannelReq struct {
	g.Meta          `path:"/sms/channels/{code}/retire" method:"post" tags:"Platform-短信" summary:"退役短信通道"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type RetireChannelRes Channel

type TestSendReq struct {
	g.Meta      `path:"/sms/test" method:"post" tags:"Platform-短信" summary:"测试发送短信"`
	ChannelCode string `json:"channelCode"`
	Phone       string `json:"phone"`
}
type TestSendRes struct {
	OK          bool   `json:"ok"`
	Detail      string `json:"detail"`
	ChannelCode string `json:"channelCode"`
	Driver      string `json:"driver"`
	Mock        bool   `json:"mock"`
	Code        string `json:"code,omitempty"`
}

type Region struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	DialCode  string    `json:"dialCode"`
	Name      string    `json:"name"`
	ISO2      string    `json:"iso2"`
	Emoji     string    `json:"emoji"`
	Sort      int       `json:"sort"`
	Enabled   bool      `json:"enabled"`
	Lifecycle string    `json:"lifecycle"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ListRegionsReq struct {
	g.Meta    `path:"/sms/regions" method:"get" tags:"Platform-短信" summary:"查询短信区域目录"`
	Keyword   string `json:"keyword" in:"query"`
	Lifecycle string `json:"lifecycle" in:"query"`
}
type ListRegionsRes []Region

type PutRegionReq struct {
	g.Meta          `path:"/sms/regions/{code}" method:"put" tags:"Platform-短信" summary:"版本化新增或修改短信区域"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
	DialCode        string `json:"dialCode"`
	Name            string `json:"name"`
	ISO2            string `json:"iso2"`
	Emoji           string `json:"emoji"`
	Sort            int    `json:"sort"`
}
type PutRegionRes Region

type EnableRegionReq struct {
	g.Meta          `path:"/sms/regions/{code}/enable" method:"post" tags:"Platform-短信" summary:"启用短信区域"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type EnableRegionRes Region
type DisableRegionReq struct {
	g.Meta          `path:"/sms/regions/{code}/disable" method:"post" tags:"Platform-短信" summary:"停用短信区域"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type DisableRegionRes Region

type RetireRegionReq struct {
	g.Meta          `path:"/sms/regions/{code}/retire" method:"post" tags:"Platform-短信" summary:"退役短信区域"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type RetireRegionRes Region

type MerchantGrant struct {
	ID           int64     `json:"id"`
	MerchantID   int64     `json:"merchantId"`
	ShopID       int64     `json:"shopId"`
	DialCodes    []string  `json:"dialCodes"`
	Unrestricted bool      `json:"unrestricted"`
	Version      int64     `json:"version"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type GetMerchantGrantReq struct {
	g.Meta     `path:"/sms/merchant-grants" method:"get" tags:"Platform-短信" summary:"查询商户短信区域开通"`
	MerchantID int64 `json:"merchantId" in:"query"`
	ShopID     int64 `json:"shopId" in:"query"`
}
type GetMerchantGrantRes MerchantGrant

type PutMerchantGrantReq struct {
	g.Meta          `path:"/sms/merchant-grants" method:"put" tags:"Platform-短信" summary:"设置商户短信区域开通"`
	CommandKey      string   `json:"commandKey"`
	ExpectedVersion int64    `json:"expectedVersion"`
	MerchantID      int64    `json:"merchantId"`
	ShopID          int64    `json:"shopId"`
	DialCodes       []string `json:"dialCodes"`
}
type PutMerchantGrantRes MerchantGrant
