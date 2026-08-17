package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/lvtuopen-ai/kernel-go/apperror"
)

type Kind string
type Driver string
type Lifecycle string
type Health string
type SecretMode string

const (
	KindRTMP Kind = "RTMP"
	KindRTC  Kind = "RTC"

	DriverStatic            Driver = "STATIC"
	DriverSRS               Driver = "SRS"
	DriverCloud             Driver = "CLOUD"
	DriverAgora             Driver = "AGORA"
	DriverAgoraMediaGateway Driver = "AGORA_MEDIA_GATEWAY"

	LifecycleActive  Lifecycle = "ACTIVE"
	LifecycleRetired Lifecycle = "RETIRED"

	HealthUnknown   Health = "UNKNOWN"
	HealthHealthy   Health = "HEALTHY"
	HealthUnhealthy Health = "UNHEALTHY"

	SecretKeep    SecretMode = "KEEP"
	SecretReplace SecretMode = "REPLACE"
	SecretClear   SecretMode = "CLEAR"
)

var (
	ErrInvalid  = apperror.New("platform.live_provider.invalid", "live provider input is invalid")
	ErrNotFound = apperror.New("platform.live_provider.not_found", "live provider was not found")
	ErrConflict = apperror.New("platform.live_provider.conflict", "live provider version or command conflicts")
	ErrRetired  = apperror.New("platform.live_provider.retired", "retired live provider cannot be changed")
	codePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
)

type Scope struct {
	Realm      string
	MerchantID int64
	Subject    string
}

func (s Scope) Valid() bool {
	return strings.TrimSpace(s.Realm) != "" && s.MerchantID >= 0 && strings.TrimSpace(s.Subject) != ""
}

type Credentials struct {
	Secret         string `json:"secret,omitempty"`
	AppCertificate string `json:"appCertificate,omitempty"`
	CustomerKey    string `json:"customerKey,omitempty"`
	CustomerSecret string `json:"customerSecret,omitempty"`
}

type CredentialSummary struct {
	SecretSet             bool
	SecretMask            string
	AppCertificateSet     bool
	AppCertificateMask    string
	CustomerCredentialSet bool
	CustomerKeyMask       string
	CustomerSecretMask    string
	KeyID                 string
}

type Provider struct {
	ID              int64
	Code            string
	Name            string
	Kind            Kind
	Driver          Driver
	App             string
	PushDomain      string
	PullDomain      string
	AgoraAppID      string
	Codec           string
	IngestDomain    string
	Region          string
	TTLSeconds      int64
	Enabled         bool
	IsDefault       bool
	Lifecycle       Lifecycle
	Health          Health
	HealthMessage   string
	HealthCheckedAt *time.Time
	Credentials     CredentialSummary
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CredentialChange struct {
	Mode           SecretMode `json:"mode"`
	Value          string     `json:"value,omitempty"`
	SecondaryValue string     `json:"secondaryValue,omitempty"`
}

type Upsert struct {
	CommandKey         string
	ExpectedVersion    int64
	Code               string
	Name               string
	Driver             Driver
	App                string
	PushDomain         string
	PullDomain         string
	AgoraAppID         string
	Codec              string
	IngestDomain       string
	Region             string
	TTLSeconds         int64
	IsDefault          bool
	Secret             CredentialChange
	AppCertificate     CredentialChange
	CustomerCredential CredentialChange
}

type Retire struct {
	CommandKey      string
	Code            string
	ExpectedVersion int64
}

type Filter struct {
	Keyword   string
	Kind      Kind
	Driver    Driver
	Lifecycle Lifecycle
}

func KindFor(driver Driver) (Kind, bool) {
	definition, found := DefinitionFor(driver)
	return definition.Kind, found
}

func NormalizeUpsert(input Upsert) Upsert {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Driver = Driver(strings.ToUpper(strings.TrimSpace(string(input.Driver))))
	input.App = strings.TrimSpace(input.App)
	input.PushDomain = strings.TrimSpace(input.PushDomain)
	input.PullDomain = strings.TrimSpace(input.PullDomain)
	input.AgoraAppID = strings.TrimSpace(input.AgoraAppID)
	input.Codec = strings.ToLower(strings.TrimSpace(input.Codec))
	input.IngestDomain = strings.TrimSpace(input.IngestDomain)
	input.Region = strings.ToLower(strings.TrimSpace(input.Region))
	if input.TTLSeconds == 0 {
		input.TTLSeconds = 7200
	}
	switch input.Driver {
	case DriverStatic:
		if input.App == "" {
			input.App = "live"
		}
		input.AgoraAppID, input.Codec, input.IngestDomain, input.Region = "", "", "", ""
		input.Secret = CredentialChange{Mode: SecretClear}
		input.AppCertificate = CredentialChange{Mode: SecretClear}
		input.CustomerCredential = CredentialChange{Mode: SecretClear}
	case DriverSRS, DriverCloud:
		if input.App == "" {
			input.App = "live"
		}
		input.AgoraAppID, input.Codec, input.IngestDomain, input.Region = "", "", "", ""
		input.AppCertificate = CredentialChange{Mode: SecretClear}
		input.CustomerCredential = CredentialChange{Mode: SecretClear}
	case DriverAgora:
		input.App, input.PushDomain, input.PullDomain = "", "", ""
		input.IngestDomain, input.Region = "", ""
		input.Secret = CredentialChange{Mode: SecretClear}
	case DriverAgoraMediaGateway:
		input.App, input.PushDomain, input.PullDomain = "", "", ""
		input.Secret = CredentialChange{Mode: SecretClear}
		if input.Region == "" {
			input.Region = "na"
		}
	}
	return input
}

func ValidateUpsert(scope Scope, input Upsert) error {
	kind, driverOK := KindFor(input.Driver)
	if !scope.Valid() || input.CommandKey == "" || len(input.CommandKey) > 64 || !codePattern.MatchString(input.Code) || input.Name == "" || len(input.Name) > 120 || !driverOK || input.ExpectedVersion < 0 || input.TTLSeconds < 60 || input.TTLSeconds > 86400*30 {
		return ErrInvalid
	}
	if !validChange(input.Secret, false) || !validChange(input.AppCertificate, false) || !validChange(input.CustomerCredential, true) {
		return ErrInvalid
	}
	if kind == KindRTMP {
		if input.App == "" || input.AgoraAppID != "" || input.Codec != "" || input.IngestDomain != "" || input.Region != "" {
			return ErrInvalid
		}
	} else {
		if input.AgoraAppID == "" || !validCodec(input.Driver, input.Codec) || input.PushDomain != "" || input.PullDomain != "" {
			return ErrInvalid
		}
		if input.Driver == DriverAgora && (input.Region != "" || input.IngestDomain != "") {
			return ErrInvalid
		}
		if input.Driver == DriverAgoraMediaGateway && !validRegion(input.Region) {
			return ErrInvalid
		}
	}
	return nil
}

func validChange(change CredentialChange, pair bool) bool {
	switch change.Mode {
	case SecretKeep:
		return change.Value == "" && change.SecondaryValue == ""
	case SecretClear:
		return change.Value == "" && change.SecondaryValue == ""
	case SecretReplace:
		if strings.TrimSpace(change.Value) == "" {
			return false
		}
		return !pair || strings.TrimSpace(change.SecondaryValue) != ""
	default:
		return false
	}
}

func validCodec(driver Driver, codec string) bool {
	if driver == DriverAgoraMediaGateway {
		return codec == "h264" || codec == "h265"
	}
	return codec == "vp8" || codec == "h264" || codec == "vp9" || codec == "av1" || codec == "h265"
}

func validRegion(region string) bool {
	return region == "na" || region == "eu" || region == "ap" || region == "cn"
}

func ValidateRetire(scope Scope, input Retire) error {
	if !scope.Valid() || strings.TrimSpace(input.CommandKey) == "" || len(input.CommandKey) > 64 || !codePattern.MatchString(input.Code) || input.ExpectedVersion <= 0 {
		return ErrInvalid
	}
	return nil
}

func RequestHash(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

type Assignment struct {
	ProviderCode string
	Name         string
	Enabled      bool
	Default      bool
}

type AssignmentSet struct {
	MerchantID int64
	Providers  []Assignment
	Version    int64
}

type PutAssignments struct {
	CommandKey      string
	ExpectedVersion int64
	MerchantID      int64
	Providers       []Assignment
}

func MaskSecret(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return ""
	}
	if len(runes) <= 10 {
		return strings.Repeat("*", len(runes))
	}
	return string(runes[:6]) + "…" + string(runes[len(runes)-4:])
}
