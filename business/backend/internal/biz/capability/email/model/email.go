package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/lvtuopen-ai/kernel-go/apperror"
)

type SecretMode string

const (
	SecretKeep    SecretMode = "KEEP"
	SecretReplace SecretMode = "REPLACE"
	SecretClear   SecretMode = "CLEAR"

	ResourceID = "singleton"
)

var (
	ErrInvalid    = apperror.New("platform.email.invalid", "email input is invalid")
	ErrNotFound   = apperror.New("platform.email.not_found", "email config was not found")
	ErrConflict   = apperror.New("platform.email.conflict", "email version or command conflicts")
	ErrNotConfigured = apperror.New("platform.email.not_configured", "email is not configured yet")
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

type Config struct {
	ID              int64
	Driver          Driver
	Enabled         bool
	PublicConfig    map[string]string
	SecretMasks     map[string]string
	CredentialKeyID string
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (c Config) Configured() bool {
	return c.Version > 0 && c.Driver != ""
}

type ConfigSecrets struct {
	Config Config
	Values map[string]string
}

type UpsertConfig struct {
	CommandKey      string
	ExpectedVersion int64
	Driver          Driver
	PublicConfig    map[string]string
	Secrets         map[string]CredentialChange
}

type SetEnabled struct {
	CommandKey      string
	ExpectedVersion int64
	Enabled         bool
}

type TestSend struct {
	To     string
	Subject string
}

type TestSendResult struct {
	OK     bool
	Detail string
	Driver Driver
	Mock   bool
}

func NormalizeUpsert(input UpsertConfig) UpsertConfig {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Driver = Driver(strings.ToLower(strings.TrimSpace(string(input.Driver))))
	input.PublicConfig = normalizePublicConfig(input.Driver, input.PublicConfig)
	secrets := make(map[string]CredentialChange, len(input.Secrets))
	for key, change := range input.Secrets {
		secrets[strings.TrimSpace(key)] = CredentialChange{Mode: SecretMode(strings.ToUpper(strings.TrimSpace(string(change.Mode)))), Value: change.Value}
	}
	input.Secrets = secrets
	return input
}

func ValidateUpsert(scope Scope, input UpsertConfig) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || input.ExpectedVersion < 0 {
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
		value := strings.TrimSpace(input.PublicConfig[field.Key])
		if field.Required && value == "" {
			return ErrInvalid
		}
		if field.Type == FieldSelect && value != "" && !validOption(field, value) {
			return ErrInvalid
		}
		if field.Type == FieldNumber && value != "" && !validPort(value) {
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
	if !scope.Valid() || !validCommandKey(strings.TrimSpace(input.CommandKey)) || input.ExpectedVersion <= 0 {
		return ErrInvalid
	}
	return nil
}

func ValidateEmail(address string) bool {
	address = strings.TrimSpace(address)
	at := strings.LastIndex(address, "@")
	if at < 1 || at >= len(address)-3 {
		return false
	}
	domain := address[at+1:]
	return strings.Contains(domain, ".") && !strings.ContainsAny(address, " \t\r\n,;")
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

func validOption(field ConfigField, value string) bool {
	for _, option := range field.Options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func validPort(value string) bool {
	if value == "" {
		return false
	}
	port := 0
	for _, item := range value {
		if item < '0' || item > '9' {
			return false
		}
		port = port*10 + int(item-'0')
		if port > 65535 {
			return false
		}
	}
	return port > 0
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
