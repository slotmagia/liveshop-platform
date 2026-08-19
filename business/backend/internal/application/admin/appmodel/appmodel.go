// Package appmodel holds the transport-neutral input and output of the admin
// surface.
package appmodel

import (
	"encoding/json"

	emailmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/email/model"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

type PutLiveProvider struct {
	Code               string
	CommandKey         string
	ExpectedVersion    int64
	Name               string
	Driver             providermodel.Driver
	App                string
	PushDomain         string
	PullDomain         string
	AgoraAppID         string
	Codec              string
	IngestDomain       string
	Region             string
	TTLSeconds         int64
	IsDefault          bool
	Secret             providermodel.CredentialChange
	AppCertificate     providermodel.CredentialChange
	CustomerCredential providermodel.CredentialChange
}

type RetireLiveProvider struct {
	Code, CommandKey string
	ExpectedVersion  int64
}

type LiveProviderAssignment struct {
	ProviderCode string `json:"providerCode"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	Default      bool   `json:"default"`
}

type PutLiveProviderAssignments struct {
	CommandKey      string
	ExpectedVersion int64
	MerchantID      int64
	Providers       []LiveProviderAssignment
}

type Activation struct {
	ModuleID string
	Version  string
}

type CapabilityCatalog struct {
	Revision uint64
	Items    []model.ModuleCapabilityCatalog
}

type PutSetting struct {
	Namespace       string
	ExpectedVersion int64
	Value           json.RawMessage
}

type PutSMSChannel struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
	Name            string
	Driver          smsmodel.Driver
	Region          string
	Priority        int
	PublicConfig    map[string]string
	Secrets         map[string]smsmodel.CredentialChange
}

type SetSMSEnabled struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
	Enabled         bool
}

type RetireSMS struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
}

type PutSMSRegion struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
	DialCode        string
	Name            string
	ISO2            string
	Emoji           string
	Sort            int
}

type PutSMSMerchantGrant struct {
	CommandKey      string
	ExpectedVersion int64
	MerchantID      int64
	ShopID          int64
	DialCodes       []string
}

type TestSMSSend struct {
	ChannelCode string
	Phone       string
}

type PutEmailConfig struct {
	CommandKey      string
	ExpectedVersion int64
	Driver          emailmodel.Driver
	PublicConfig    map[string]string
	Secrets         map[string]emailmodel.CredentialChange
}

type SetEmailEnabled struct {
	CommandKey      string
	ExpectedVersion int64
	Enabled         bool
}

type TestEmailSend struct {
	To      string
	Subject string
}

type PutStorageChannel struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
	Name            string
	Driver          storagemodel.Driver
	PublicConfig    map[string]string
	Secrets         map[string]storagemodel.CredentialChange
}

type SetStorageEnabled struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
	Enabled         bool
}

type SetStorageDefault struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
}

type RetireStorage struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
}

type TestStorageChannel struct {
	Code string
}

type ReplaceNotifyPolicy struct {
	EventKey        string
	CommandKey      string
	ExpectedVersion int64
	DispatchMode    string
	DelaySeconds    int
	Channels        map[string]NotifyChannelPolicy
}

type NotifyChannelPolicy struct {
	Enabled      bool
	TemplateCode string
}

type UpsertNotifyTemplate struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
	Channel         string
	TextTemplate    string
	Subject         string
	BodyHTML        string
	Title           string
	Body            string
	Variables       []string
}

type RetireNotifyTemplate struct {
	Code            string
	CommandKey      string
	ExpectedVersion int64
}

type ReplaceNotifyInApp struct {
	CommandKey      string
	ExpectedVersion int64
	Enabled         bool
}

type PutI18nConfig struct {
	CommandKey      string
	ExpectedVersion int64
	Provider        string
	APIKey          string
	APIKeyClear     bool
}

type PublishI18nText struct {
	CommandKey      string
	ExpectedVersion int64
	EntityType      string
	EntityID        string
	Locale          string
	Value           string
	MerchantID      int64
	ShopID          int64
}

type FillI18nTexts struct {
	CommandKey string
	EntityType string
	Locale     string
}
