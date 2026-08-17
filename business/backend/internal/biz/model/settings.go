package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/lvtuopen-ai/kernel-go/apperror"
	"github.com/lvtuopen-ai/kernel-go/principal"
)

var (
	ErrSettingsInvalid  = apperror.New("platform.settings.invalid", "settings namespace or value is invalid")
	ErrSettingsConflict = apperror.New("platform.settings.conflict", "settings version conflict")
)

var settingNamespacePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
var settingSecretKeyPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|private.?key|credential|api.?key)`)

type SettingScope struct {
	Realm      string
	MerchantID int64
	Subject    string
}

// Valid accepts back-office realms and the merchant tenancy Identity actually
// mints. Platform operators have no shop context, so their documents are
// keyed by realm plus merchant_id=0. Merchant scopes require a real merchant.
func (s SettingScope) Valid() bool {
	realm, known := principal.ParseRealm(s.Realm)
	if !known || s.Subject == "" {
		return false
	}
	switch realm {
	case principal.RealmPlatform:
		return s.MerchantID == 0
	case principal.RealmMerchant:
		return s.MerchantID > 0
	default:
		return false
	}
}

type SettingDocument struct {
	Namespace string
	Value     json.RawMessage
	Version   int64
	UpdatedBy string
	UpdatedAt time.Time
}

func NormalizeNamespace(namespace string) string {
	return strings.ToLower(strings.TrimSpace(namespace))
}

func ValidNamespace(namespace string) bool {
	return settingNamespacePattern.MatchString(namespace)
}

// CanonicalSettingValue rejects secret-looking material and returns the
// canonical JSON encoding stored for the namespace.
func CanonicalSettingValue(namespace string, value json.RawMessage) ([]byte, error) {
	if namespace == "secrets" || len(value) == 0 {
		return nil, ErrSettingsInvalid
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, ErrSettingsInvalid
	}
	if _, ok := decoded.(map[string]any); !ok || containsSecret(decoded) {
		return nil, ErrSettingsInvalid
	}
	return json.Marshal(decoded)
}

func containsSecret(value any) bool {
	switch current := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if settingSecretKeyPattern.MatchString(key) || containsSecret(current[key]) {
				return true
			}
		}
	case []any:
		for _, item := range current {
			if containsSecret(item) {
				return true
			}
		}
	}
	return false
}

func CompactJSON(value []byte) []byte {
	var output bytes.Buffer
	if json.Compact(&output, value) != nil {
		return value
	}
	return output.Bytes()
}
