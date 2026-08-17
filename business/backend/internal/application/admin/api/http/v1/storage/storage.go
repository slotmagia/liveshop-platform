// Package storage defines the Admin HTTP wire contract for Platform object storage.
// Secret values are write-only.
package storage

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
	g.Meta `path:"/storage/drivers" method:"get" tags:"Platform-存储" summary:"查询存储驱动及表单元数据"`
}
type DriversRes []DriverDefinition

type Channel struct {
	ID              int64             `json:"id"`
	Code            string            `json:"code"`
	Name            string            `json:"name"`
	Driver          string            `json:"driver"`
	Enabled         bool              `json:"enabled"`
	IsDefault       bool              `json:"isDefault"`
	Lifecycle       string            `json:"lifecycle"`
	PublicConfig    map[string]string `json:"publicConfig"`
	SecretMasks     map[string]string `json:"secretMasks"`
	CredentialKeyID string            `json:"credentialKeyId,omitempty"`
	Version         int64             `json:"version"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type ListChannelsReq struct {
	g.Meta    `path:"/storage/channels" method:"get" tags:"Platform-存储" summary:"查询存储通道目录"`
	Keyword   string `json:"keyword" in:"query"`
	Driver    string `json:"driver" in:"query"`
	Lifecycle string `json:"lifecycle" in:"query"`
}
type ListChannelsRes []Channel

type PutChannelReq struct {
	g.Meta          `path:"/storage/channels/{code}" method:"put" tags:"Platform-存储" summary:"版本化新增或修改存储通道"`
	Code            string                      `json:"code" in:"path"`
	CommandKey      string                      `json:"commandKey"`
	ExpectedVersion int64                       `json:"expectedVersion"`
	Name            string                      `json:"name"`
	Driver          string                      `json:"driver"`
	PublicConfig    map[string]string           `json:"publicConfig"`
	Secrets         map[string]CredentialChange `json:"secrets"`
}
type PutChannelRes Channel

type EnableChannelReq struct {
	g.Meta          `path:"/storage/channels/{code}/enable" method:"post" tags:"Platform-存储" summary:"启用存储通道"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type EnableChannelRes Channel
type DisableChannelReq struct {
	g.Meta          `path:"/storage/channels/{code}/disable" method:"post" tags:"Platform-存储" summary:"停用存储通道"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type DisableChannelRes Channel

type SetDefaultReq struct {
	g.Meta          `path:"/storage/channels/{code}/default" method:"post" tags:"Platform-存储" summary:"将存储通道设为默认"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type SetDefaultRes Channel

type RetireChannelReq struct {
	g.Meta          `path:"/storage/channels/{code}/retire" method:"post" tags:"Platform-存储" summary:"退役存储通道"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type RetireChannelRes Channel

type TestChannelReq struct {
	g.Meta `path:"/storage/channels/{code}/test" method:"post" tags:"Platform-存储" summary:"测试写入存储通道"`
	Code   string `json:"code" in:"path"`
}
type TestChannelRes struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	URL    string `json:"url,omitempty"`
	Driver string `json:"driver"`
}
