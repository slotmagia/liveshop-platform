package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
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

	WildcardRegion = "*"
)

var (
	ErrInvalid   = apperror.New("platform.sms.invalid", "sms input is invalid")
	ErrNotFound  = apperror.New("platform.sms.not_found", "sms resource was not found")
	ErrConflict  = apperror.New("platform.sms.conflict", "sms version or command conflicts")
	ErrRetired   = apperror.New("platform.sms.retired", "retired sms resource cannot be changed")
	ErrNoChannel = apperror.New("platform.sms.no_channel", "no enabled sms channel matches the phone")
	ErrInUse     = apperror.New("platform.sms.in_use", "sms region is still referenced by an active channel")

	channelCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	regionCodePattern  = regexp.MustCompile(`^[1-9][0-9]{0,6}$`)
	iso2Pattern        = regexp.MustCompile(`^[A-Z]{2}$`)
	dialCodePattern    = regexp.MustCompile(`^\+[1-9][0-9]{0,6}$`)
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
	Region          string
	Priority        int
	Enabled         bool
	Lifecycle       Lifecycle
	PublicConfig    map[string]string
	SecretMasks     map[string]string
	CredentialKeyID string
	Version         int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Region struct {
	ID        int64
	Code      string
	DialCode  string
	Name      string
	ISO2      string
	Emoji     string
	Sort      int
	Enabled   bool
	Lifecycle Lifecycle
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MerchantGrant struct {
	ID           int64
	MerchantID   int64
	ShopID       int64
	DialCodes    []string
	Unrestricted bool
	Version      int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ChannelFilter struct {
	Keyword   string
	Driver    Driver
	Lifecycle Lifecycle
}

type RegionFilter struct {
	Keyword   string
	Lifecycle Lifecycle
}

type UpsertChannel struct {
	CommandKey      string
	ExpectedVersion int64
	Code            string
	Name            string
	Driver          Driver
	Region          string
	Priority        int
	PublicConfig    map[string]string
	Secrets         map[string]CredentialChange
}

type SetEnabled struct {
	CommandKey      string
	ExpectedVersion int64
	Code            string
	Enabled         bool
}

type Retire struct {
	CommandKey      string
	ExpectedVersion int64
	Code            string
}

type UpsertRegion struct {
	CommandKey      string
	ExpectedVersion int64
	Code            string
	DialCode        string
	Name            string
	ISO2            string
	Emoji           string
	Sort            int
}

type PutMerchantGrant struct {
	CommandKey      string
	ExpectedVersion int64
	MerchantID      int64
	ShopID          int64
	DialCodes       []string
}

type TestSend struct {
	ChannelCode string
	Phone       string
}

type TestSendResult struct {
	OK          bool
	Detail      string
	ChannelCode string
	Driver      Driver
	Mock        bool
	Code        string
}

type ChannelSecrets struct {
	Channel Channel
	Config  map[string]string
}

func NormalizeUpsertChannel(input UpsertChannel) UpsertChannel {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Driver = Driver(strings.ToLower(strings.TrimSpace(string(input.Driver))))
	input.Region = strings.TrimSpace(input.Region)
	if input.Region == "" {
		input.Region = WildcardRegion
	}
	input.PublicConfig = normalizePublicConfig(input.Driver, input.PublicConfig)
	secrets := make(map[string]CredentialChange, len(input.Secrets))
	for key, change := range input.Secrets {
		secrets[strings.TrimSpace(key)] = CredentialChange{Mode: SecretMode(strings.ToUpper(strings.TrimSpace(string(change.Mode)))), Value: change.Value}
	}
	input.Secrets = secrets
	return input
}

func NormalizeUpsertRegion(input UpsertRegion) UpsertRegion {
	input.CommandKey = strings.TrimSpace(input.CommandKey)
	input.DialCode = strings.TrimSpace(input.DialCode)
	input.Name = strings.TrimSpace(input.Name)
	input.ISO2 = strings.ToUpper(strings.TrimSpace(input.ISO2))
	input.Emoji = strings.TrimSpace(input.Emoji)
	if input.Code == "" {
		input.Code = strings.TrimPrefix(input.DialCode, "+")
	}
	input.Code = strings.TrimSpace(input.Code)
	return input
}

func ValidateUpsertChannel(scope Scope, input UpsertChannel) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || !channelCodePattern.MatchString(input.Code) || input.Name == "" || len(input.Name) > 120 || input.ExpectedVersion < 0 || input.Priority < 0 || input.Priority > 1000 {
		return ErrInvalid
	}
	definition, ok := DefinitionFor(input.Driver)
	if !ok {
		return ErrInvalid
	}
	if input.Region != WildcardRegion && !dialCodePattern.MatchString(input.Region) {
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
	if !scope.Valid() || !validCommandKey(input.CommandKey) || !channelCodePattern.MatchString(input.Code) && !regionCodePattern.MatchString(input.Code) || input.ExpectedVersion <= 0 {
		return ErrInvalid
	}
	return nil
}

func ValidateRetire(scope Scope, input Retire, region bool) error {
	pattern := channelCodePattern
	if region {
		pattern = regionCodePattern
	}
	if !scope.Valid() || !validCommandKey(input.CommandKey) || !pattern.MatchString(input.Code) || input.ExpectedVersion <= 0 {
		return ErrInvalid
	}
	return nil
}

func ValidateUpsertRegion(scope Scope, input UpsertRegion) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || !regionCodePattern.MatchString(input.Code) || !dialCodePattern.MatchString(input.DialCode) || input.Name == "" || len(input.Name) > 64 || !iso2Pattern.MatchString(input.ISO2) || input.ExpectedVersion < 0 || input.Sort < 0 || input.Sort > 10000 || len(input.Emoji) > 16 {
		return ErrInvalid
	}
	if input.Code != strings.TrimPrefix(input.DialCode, "+") {
		return ErrInvalid
	}
	return nil
}

func ValidatePutGrant(scope Scope, input PutMerchantGrant) error {
	if !scope.Valid() || !validCommandKey(input.CommandKey) || input.MerchantID <= 0 || input.ShopID <= 0 || input.ExpectedVersion < 0 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, dial := range input.DialCodes {
		dial = strings.TrimSpace(dial)
		if !dialCodePattern.MatchString(dial) || seen[dial] {
			return ErrInvalid
		}
		seen[dial] = true
	}
	return nil
}

func ValidatePhone(phone string) bool {
	phone = strings.TrimSpace(phone)
	if !strings.HasPrefix(phone, "+") || len(phone) < 8 {
		return false
	}
	for _, item := range phone[1:] {
		if item < '0' || item > '9' {
			return false
		}
	}
	return true
}

func RouteChannels(phone string, channels []Channel) []Channel {
	candidates := make([]Channel, 0)
	for _, channel := range channels {
		if channel.Lifecycle != LifecycleActive || !channel.Enabled {
			continue
		}
		if channel.Region == WildcardRegion || channel.Region == "" || strings.HasPrefix(phone, channel.Region) {
			candidates = append(candidates, channel)
		}
	}
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if lessSpecific(candidates[j], candidates[i]) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	return candidates
}

func lessSpecific(left, right Channel) bool {
	ls, rs := specificity(left.Region), specificity(right.Region)
	if ls != rs {
		return ls > rs
	}
	if left.Priority != right.Priority {
		return left.Priority > right.Priority
	}
	return left.Code < right.Code
}

func specificity(region string) int {
	if region == WildcardRegion || region == "" {
		return 0
	}
	return len(region)
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

func GrantResourceID(merchantID, shopID int64) string {
	return strconv.FormatInt(merchantID, 10) + ":" + strconv.FormatInt(shopID, 10)
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
