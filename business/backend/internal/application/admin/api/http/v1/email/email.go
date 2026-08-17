// Package email defines the Admin HTTP wire contract for Platform email.
// Secret values are write-only. Templates belong to notification events.
package email

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type CredentialChange struct {
	Mode  string `json:"mode"`
	Value string `json:"value,omitempty"`
}

type DriverFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type DriverField struct {
	Key         string              `json:"key"`
	Label       string              `json:"label"`
	Type        string              `json:"type"`
	Required    bool                `json:"required"`
	Secret      bool                `json:"secret"`
	Placeholder string              `json:"placeholder,omitempty"`
	Help        string              `json:"help,omitempty"`
	Options     []DriverFieldOption `json:"options,omitempty"`
}

type DriverDefinition struct {
	Code        string        `json:"code"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Fields      []DriverField `json:"fields"`
}

type DriversReq struct {
	g.Meta `path:"/email/drivers" method:"get" tags:"Platform-邮件" summary:"查询邮件驱动及表单元数据"`
}
type DriversRes []DriverDefinition

type Config struct {
	ID              int64             `json:"id"`
	Driver          string            `json:"driver"`
	Enabled         bool              `json:"enabled"`
	PublicConfig    map[string]string `json:"publicConfig"`
	SecretMasks     map[string]string `json:"secretMasks"`
	CredentialKeyID string            `json:"credentialKeyId,omitempty"`
	Version         int64             `json:"version"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type GetConfigReq struct {
	g.Meta `path:"/email/config" method:"get" tags:"Platform-邮件" summary:"取当前邮件发信配置"`
}
type GetConfigRes Config

type PutConfigReq struct {
	g.Meta          `path:"/email/config" method:"put" tags:"Platform-邮件" summary:"版本化保存邮件发信配置"`
	CommandKey      string                      `json:"commandKey"`
	ExpectedVersion int64                       `json:"expectedVersion"`
	Driver          string                      `json:"driver"`
	PublicConfig    map[string]string           `json:"publicConfig"`
	Secrets         map[string]CredentialChange `json:"secrets"`
}
type PutConfigRes Config

type EnableReq struct {
	g.Meta          `path:"/email/config/enable" method:"post" tags:"Platform-邮件" summary:"启用邮件发信"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type EnableRes Config
type DisableReq struct {
	g.Meta          `path:"/email/config/disable" method:"post" tags:"Platform-邮件" summary:"停用邮件发信"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type DisableRes Config

type TestSendReq struct {
	g.Meta  `path:"/email/config/test" method:"post" tags:"Platform-邮件" summary:"发一封测试邮件"`
	To      string `json:"to"`
	Subject string `json:"subject"`
}
type TestSendRes struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Driver string `json:"driver"`
	Mock   bool   `json:"mock"`
}
