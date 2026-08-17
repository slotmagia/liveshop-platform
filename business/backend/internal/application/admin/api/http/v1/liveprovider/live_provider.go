// Package liveprovider defines the Admin HTTP wire contract for the Platform
// live-provider catalogue. Secret values are write-only.
package liveprovider

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
)

type CredentialChange struct {
	Mode           string `json:"mode"`
	Value          string `json:"value,omitempty"`
	SecondaryValue string `json:"secondaryValue,omitempty"`
}

type DriverOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type DriverField struct {
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Group       string         `json:"group,omitempty"`
	Required    bool           `json:"required"`
	Secret      bool           `json:"secret"`
	Credential  string         `json:"credential,omitempty"`
	Default     string         `json:"default,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
	Help        string         `json:"help,omitempty"`
	Options     []DriverOption `json:"options,omitempty"`
	Min         int64          `json:"min,omitempty"`
	Max         int64          `json:"max,omitempty"`
	Advanced    bool           `json:"advanced"`
}

type DriverDefinition struct {
	Code          string        `json:"code"`
	Name          string        `json:"name"`
	Kind          string        `json:"kind"`
	PushTransport string        `json:"pushTransport"`
	Description   string        `json:"description"`
	Fields        []DriverField `json:"fields"`
}

type DriversReq struct {
	g.Meta `path:"/live-providers/drivers" method:"get" tags:"Platform-流媒体方式" summary:"查询流媒体驱动及表单元数据"`
}
type DriversRes []DriverDefinition

type Provider struct {
	ID                    int64      `json:"id"`
	Code                  string     `json:"code"`
	Name                  string     `json:"name"`
	Kind                  string     `json:"kind"`
	Driver                string     `json:"driver"`
	App                   string     `json:"app"`
	PushDomain            string     `json:"pushDomain"`
	PullDomain            string     `json:"pullDomain"`
	AgoraAppID            string     `json:"agoraAppId"`
	Codec                 string     `json:"codec"`
	IngestDomain          string     `json:"ingestDomain"`
	Region                string     `json:"region"`
	TTLSeconds            int64      `json:"ttlSeconds"`
	Enabled               bool       `json:"enabled"`
	IsDefault             bool       `json:"isDefault"`
	Lifecycle             string     `json:"lifecycle"`
	HealthStatus          string     `json:"healthStatus"`
	HealthMessage         string     `json:"healthMessage,omitempty"`
	HealthCheckedAt       *time.Time `json:"healthCheckedAt,omitempty"`
	SecretSet             bool       `json:"secretSet"`
	SecretMask            string     `json:"secretMask,omitempty"`
	AppCertificateSet     bool       `json:"appCertificateSet"`
	AppCertificateMask    string     `json:"appCertificateMask,omitempty"`
	CustomerCredentialSet bool       `json:"customerCredentialSet"`
	CustomerKeyMask       string     `json:"customerKeyMask,omitempty"`
	CustomerSecretMask    string     `json:"customerSecretMask,omitempty"`
	CredentialKeyID       string     `json:"credentialKeyId,omitempty"`
	Version               int64      `json:"version"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type ListReq struct {
	g.Meta    `path:"/live-providers" method:"get" tags:"Platform-流媒体方式" summary:"查询流媒体方式目录"`
	Keyword   string `json:"keyword" in:"query"`
	Kind      string `json:"kind" in:"query"`
	Driver    string `json:"driver" in:"query"`
	Lifecycle string `json:"lifecycle" in:"query"`
}
type ListRes []Provider

type PutReq struct {
	g.Meta             `path:"/live-providers/{code}" method:"put" tags:"Platform-流媒体方式" summary:"版本化新增或修改流媒体方式"`
	Code               string           `json:"code" in:"path"`
	CommandKey         string           `json:"commandKey"`
	ExpectedVersion    int64            `json:"expectedVersion"`
	Name               string           `json:"name"`
	Driver             string           `json:"driver"`
	App                string           `json:"app"`
	PushDomain         string           `json:"pushDomain"`
	PullDomain         string           `json:"pullDomain"`
	AgoraAppID         string           `json:"agoraAppId"`
	Codec              string           `json:"codec"`
	IngestDomain       string           `json:"ingestDomain"`
	Region             string           `json:"region"`
	TTLSeconds         int64            `json:"ttlSeconds"`
	IsDefault          bool             `json:"isDefault"`
	Secret             CredentialChange `json:"secret"`
	AppCertificate     CredentialChange `json:"appCertificate"`
	CustomerCredential CredentialChange `json:"customerCredential"`
}
type PutRes Provider

type RetireReq struct {
	g.Meta          `path:"/live-providers/{code}/retire" method:"post" tags:"Platform-流媒体方式" summary:"退役流媒体方式"`
	Code            string `json:"code" in:"path"`
	CommandKey      string `json:"commandKey"`
	ExpectedVersion int64  `json:"expectedVersion"`
}
type RetireRes Provider

type Assignment struct {
	ProviderCode string `json:"providerCode"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	Default      bool   `json:"default"`
}
type AssignmentSet struct {
	MerchantID int64        `json:"merchantId"`
	Providers  []Assignment `json:"providers"`
	Version    int64        `json:"version"`
}
type GetAssignmentsReq struct {
	g.Meta     `path:"/live-providers/assignments" method:"get" tags:"Platform-流媒体方式" summary:"查询商户流媒体方式授权"`
	MerchantID int64 `json:"merchantId" in:"query"`
}
type GetAssignmentsRes AssignmentSet
type PutAssignmentsReq struct {
	g.Meta          `path:"/live-providers/assignments" method:"put" tags:"Platform-流媒体方式" summary:"替换商户流媒体方式授权"`
	CommandKey      string       `json:"commandKey"`
	ExpectedVersion int64        `json:"expectedVersion"`
	MerchantID      int64        `json:"merchantId"`
	Providers       []Assignment `json:"providers"`
}
type PutAssignmentsRes AssignmentSet
