package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/lvtuopen-ai/kernel-go/apperror"
)

const (
	ProviderNoop   Provider = "noop"
	ProviderDeepL  Provider = "deepl"
	ProviderGoogle Provider = "google"

	StatusMachine   = "machine"
	StatusPublished = "published"
	SourceHuman     = "human"
	SourceMachine   = "machine"

	SourceLocale = "zh-CN"
)

var (
	ErrInvalid       = apperror.New("platform.i18n.invalid", "i18n input is invalid")
	ErrNotFound      = apperror.New("platform.i18n.not_found", "i18n fact was not found")
	ErrConflict      = apperror.New("platform.i18n.conflict", "i18n version or command conflicts")
	ErrEntityUnknown = apperror.New("platform.i18n.entity_unknown", "unknown translatable entity type")
	ErrLocaleUnknown = apperror.New("platform.i18n.locale_unknown", "locale is not a platform target language")
	ErrProviderKey   = apperror.New("platform.i18n.provider_key", "machine translation provider key is not set")
	ErrUnavailable   = apperror.New("platform.i18n.unavailable", "i18n store is unavailable")
)

type Provider string

type Scope struct {
	Realm      string
	MerchantID int64
	Subject    string
}

func (s Scope) Valid() bool {
	return strings.TrimSpace(s.Realm) != "" && s.MerchantID >= 0 && strings.TrimSpace(s.Subject) != ""
}

type DriverField struct {
	Key         string
	Label       string
	Type        string
	Required    bool
	Secret      bool
	Placeholder string
	Help        string
}

type DriverDefinition struct {
	Code        string
	Name        string
	Description string
	Fields      []DriverField
}

func DriverDefinitions() []DriverDefinition {
	apiKey := DriverField{Key: "apiKey", Label: "API Key", Type: "PASSWORD", Required: false, Secret: true, Help: "空值保留已保存密钥"}
	return []DriverDefinition{
		{Code: string(ProviderNoop), Name: "No-op", Description: "把源文复制为机器草稿，不调用外部服务", Fields: nil},
		{Code: string(ProviderDeepL), Name: "DeepL", Description: "DeepL 机器翻译", Fields: []DriverField{apiKey}},
		{Code: string(ProviderGoogle), Name: "Google", Description: "Google Cloud Translation", Fields: []DriverField{apiKey}},
	}
}

func ValidProvider(value Provider) bool {
	switch value {
	case ProviderNoop, ProviderDeepL, ProviderGoogle:
		return true
	default:
		return false
	}
}

type Locale struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

func PlatformLocales() []Locale {
	return []Locale{{Code: "en-US", Label: "English"}}
}

func IsTargetLocale(code string) bool {
	code = strings.TrimSpace(code)
	for _, item := range PlatformLocales() {
		if item.Code == code {
			return true
		}
	}
	return false
}

type Entity struct {
	EntityType  string `json:"entityType"`
	Label       string `json:"label"`
	OwnerModule string `json:"ownerModule"`
	Field       string `json:"field"`
}

func Entities() []Entity {
	return []Entity{
		{EntityType: "catalog.category", Label: "商品类目", OwnerModule: "catalog", Field: "name"},
		{EntityType: "live.gift", Label: "直播礼物", OwnerModule: "live", Field: "name"},
		{EntityType: "trade.payment.channel", Label: "支付通道", OwnerModule: "trade", Field: "name"},
	}
}

func KnownEntity(entityType string) bool {
	for _, item := range Entities() {
		if item.EntityType == entityType {
			return true
		}
	}
	return false
}

type Config struct {
	Provider  Provider
	APIKeySet bool
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UpsertConfig struct {
	CommandKey      string
	ExpectedVersion int64
	Provider        Provider
	APIKey          string
	APIKeyClear     bool
}

type WorklistRow struct {
	EntityID   string
	MerchantID int64
	ShopID     int64
	Source     string
	Value      string
	Status     string
	TextSource string
	Stale      bool
	Version    int64
}

type PublishInput struct {
	CommandKey      string
	ExpectedVersion int64
	EntityType      string
	EntityID        string
	Locale          string
	Value           string
	MerchantID      int64
	ShopID          int64
}

type PublishResult struct {
	OK      bool
	Version int64
}

type FillInput struct {
	CommandKey string
	EntityType string
	Locale     string
}

type FillResult struct {
	Provider Provider
	Filled   int
	Skipped  int
}

type SourceSnapshot struct {
	EntityType    string
	EntityID      string
	MerchantID    int64
	ShopID        int64
	Source        string
	SourceVersion int64
}

type PublishedText struct {
	EntityID string
	Value    string
	Version  int64
}

func NormalizeLocale(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return SourceLocale
	}
	lower := strings.ToLower(strings.ReplaceAll(value, "_", "-"))
	if strings.HasPrefix(lower, "zh") {
		return SourceLocale
	}
	if strings.HasPrefix(lower, "en") {
		return "en-US"
	}
	return SourceLocale
}

func RequestHash(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func ValidCommandKey(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 64
}
