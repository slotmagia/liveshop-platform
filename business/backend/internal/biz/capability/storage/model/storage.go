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

type Lifecycle string
type SecretMode string

const (
	LifecycleActive  Lifecycle = "ACTIVE"
	LifecycleRetired Lifecycle = "RETIRED"

	SecretKeep    SecretMode = "KEEP"
	SecretReplace SecretMode = "REPLACE"
	SecretClear   SecretMode = "CLEAR"
)

var (
	ErrInvalid  = apperror.New("platform.storage.invalid", "storage input is invalid")
	ErrNotFound = apperror.New("platform.storage.not_found", "storage channel was not found")
	ErrConflict = apperror.New("platform.storage.conflict", "storage version or command conflicts")
	ErrRetired  = apperror.New("platform.storage.retired", "retired storage channel cannot be changed")
	ErrDisabled = apperror.New("platform.storage.disabled", "disabled storage channel cannot be the default")

	channelCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
)

type Scope struct {
	Realm      string
	MerchantID int64
	Subject    string
}

func (s Scope) Valid() bool {
	return strings.TrimSpace(s.Realm) != "" && s.MerchantID >= 0 && strings.TrimSpace(s.Subject) != ""
}

type CredentialChange struct {
	Mode  SecretMode
	Value string
}

type Channel struct {
	ID              int64
	Code            string
	Name            string
	Driver          Driver
	Enabled         bool
	IsDefault       bool
	Lifecycle       Lifecycle
	PublicConfig    map[string]string
	SecretMasks     map[string]string
	CredentialKeyID string
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ChannelFilter struct {
	Keyword   string
	Driver    Driver
	Lifecycle Lifecycle
}

type UpsertChannel struct {
	CommandKey      string
	ExpectedVersion int64
	Code            string
	Name            string
	Driver          Driver
	PublicConfig    map[string]string
	Secrets         map[string]CredentialChange
}

type SetEnabled struct {
	CommandKey      string
	ExpectedVersion int64
	Code            string
	Enabled         bool
}

type SetDefault struct {
	CommandKey      string
	ExpectedVersion int64
	Code            string
}

type Retire struct {
	CommandKey      string
	ExpectedVersion int64
	Code            string
}

type TestChannel struct {
	Code string
}

type TestResult struct {
	OK     bool
	Detail string
	URL    string
	Driver Driver
}

type Object struct {
	Key         string
	ContentType string
	Content     string
}

type ChannelSecrets struct {
	Channel Channel
	Config  map[string]string
}

func NormalizeUpsert(input UpsertChannel) UpsertChannel {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Driver = Driver(strings.ToLower(strings.TrimSpace(string(input.Driver))))
	input.PublicConfig = normalizePublicConfig(input.Driver, input.PublicConfig)
	secrets := make(map[string]CredentialChange, len(input.Secrets))
	for key, change := range input.Secrets {
		secrets[strings.TrimSpace(key)] = CredentialChange{Mode: SecretMode(strings.ToUpper(strings.TrimSpace(string(change.Mode)))), Value: change.Value}
	}
	input.Secrets = secrets
	return input
}

func ValidateUpsert(scope Scope, input UpsertChannel) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || !channelCodePattern.MatchString(input.Code) || input.Name == "" || len(input.Name) > 120 || input.ExpectedVersion < 0 {
		return ErrInvalid
	}
	definition, ok := DefinitionFor(input.Driver)
	if !ok {
		return ErrInvalid
	}
	publicKeys := map[string]bool{}
	for _, field := range definition.Fields {
		if field.Secret {
			continue
		}
		publicKeys[field.Key] = true
		if field.Required && strings.TrimSpace(input.PublicConfig[field.Key]) == "" {
			return ErrInvalid
		}
	}
	for key := range input.PublicConfig {
		if !publicKeys[key] {
			return ErrInvalid
		}
	}
	secretKeys := map[string]bool{}
	for _, key := range SecretFieldKeys(input.Driver) {
		secretKeys[key] = true
	}
	for key, change := range input.Secrets {
		if !secretKeys[key] || !validChange(change) {
			return ErrInvalid
		}
	}
	for key := range secretKeys {
		if _, ok := input.Secrets[key]; !ok {
			return ErrInvalid
		}
	}
	return nil
}

func ValidateSetEnabled(scope Scope, input SetEnabled) error {
	if !scope.Valid() || !validCommandKey(strings.TrimSpace(input.CommandKey)) || !channelCodePattern.MatchString(strings.ToLower(strings.TrimSpace(input.Code))) || input.ExpectedVersion <= 0 {
		return ErrInvalid
	}
	return nil
}

func ValidateSetDefault(scope Scope, input SetDefault) error {
	if !scope.Valid() || !validCommandKey(strings.TrimSpace(input.CommandKey)) || !channelCodePattern.MatchString(strings.ToLower(strings.TrimSpace(input.Code))) || input.ExpectedVersion <= 0 {
		return ErrInvalid
	}
	return nil
}

func ValidateRetire(scope Scope, input Retire) error {
	if !scope.Valid() || !validCommandKey(strings.TrimSpace(input.CommandKey)) || !channelCodePattern.MatchString(strings.ToLower(strings.TrimSpace(input.Code))) || input.ExpectedVersion <= 0 {
		return ErrInvalid
	}
	return nil
}

func ApplySecrets(current map[string]string, changes map[string]CredentialChange) map[string]string {
	next := make(map[string]string, len(current)+len(changes))
	for key, value := range current {
		next[key] = value
	}
	for key, change := range changes {
		switch change.Mode {
		case SecretReplace:
			next[key] = change.Value
		case SecretClear:
			delete(next, key)
		}
	}
	return next
}

func MaskSecrets(values map[string]string) map[string]string {
	masks := make(map[string]string, len(values))
	for key, value := range values {
		masks[key] = MaskSecret(value)
	}
	return masks
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

func RequestHash(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func validCommandKey(value string) bool {
	return value != "" && len(value) <= 64
}

func validChange(change CredentialChange) bool {
	switch change.Mode {
	case SecretKeep, SecretClear:
		return change.Value == ""
	case SecretReplace:
		return strings.TrimSpace(change.Value) != ""
	default:
		return false
	}
}

func normalizePublicConfig(driver Driver, input map[string]string) map[string]string {
	definition, ok := DefinitionFor(driver)
	if !ok {
		return map[string]string{}
	}
	output := map[string]string{}
	for _, field := range definition.Fields {
		if field.Secret {
			continue
		}
		if value := strings.TrimSpace(input[field.Key]); value != "" {
			output[field.Key] = value
		}
	}
	return output
}
