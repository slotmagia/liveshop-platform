package model

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Setting field types match Host form fields. The catalog is owned by Platform
// and never includes other modules' facts (live/trade/identity).
const (
	SettingFieldText     = "TEXT"
	SettingFieldNumber   = "NUMBER"
	SettingFieldBool     = "BOOL"
	SettingFieldSelect   = "SELECT"
	SettingFieldTextarea = "TEXTAREA"
)

type SettingFieldOption struct {
	Value string
	Label string
}

type SettingField struct {
	Key         string
	Label       string
	Type        string
	Help        string
	Placeholder string
	Options     []SettingFieldOption
}

type SettingCategory struct {
	Key        string
	Label      string
	GroupKey   string
	GroupLabel string
	Fields     []SettingField
}

type SettingGroup struct {
	Key        string
	Label      string
	Categories []SettingCategory
}

func SettingCatalog() []SettingGroup {
	return []SettingGroup{
		{
			Key: "basic", Label: "基础设置",
			Categories: []SettingCategory{{
				Key: "site", Label: "站点信息", GroupKey: "basic", GroupLabel: "基础设置",
				Fields: []SettingField{
					{Key: "site_name", Label: "站点名称", Type: SettingFieldText, Help: "C 端浏览器标题与页脚展示", Placeholder: "直播商城"},
					{Key: "customer_phone", Label: "客服电话", Type: SettingFieldText, Help: "C 端客服联系方式", Placeholder: "400-000-0000"},
					{Key: "icp", Label: "ICP 备案号", Type: SettingFieldText, Help: "页脚备案信息"},
				},
			}},
		},
		{
			Key: "domain", Label: "域名配置",
			Categories: []SettingCategory{{
				Key: "domain-base", Label: "主域名", GroupKey: "domain", GroupLabel: "域名配置",
				Fields: []SettingField{
					{Key: "root_domain", Label: "平台主域名", Type: SettingFieldText, Help: "店铺/直播间子域名挂在其下（*.shop / *.live）", Placeholder: "wopays.com"},
					{Key: "shop_domain", Label: "商城域名", Type: SettingFieldText, Help: "C 端商城访问域名", Placeholder: "shop.wopays.com"},
					{Key: "live_domain", Label: "直播间域名", Type: SettingFieldText, Help: "直播观众端访问域名", Placeholder: "live.wopays.com"},
					{Key: "rts_domain", Label: "推流端域名", Type: SettingFieldText, Help: "主播推流端访问域名", Placeholder: "rts.wopays.com"},
					{Key: "admin_domain", Label: "总后台域名", Type: SettingFieldText, Help: "平台运营后台访问域名", Placeholder: "adminer.wopays.com"},
					{Key: "merchant_domain", Label: "商户后台域名", Type: SettingFieldText, Help: "商户运营后台访问域名", Placeholder: "merchant.wopays.com"},
					{Key: "force_https", Label: "强制 HTTPS", Type: SettingFieldBool, Help: "C 端是否强制跳转 HTTPS", Placeholder: "true"},
				},
			}},
		},
	}
}

func SettingCategoryByKey(namespace string) (SettingCategory, bool) {
	namespace = NormalizeNamespace(namespace)
	for _, group := range SettingCatalog() {
		for _, category := range group.Categories {
			if category.Key == namespace {
				return category, true
			}
		}
	}
	return SettingCategory{}, false
}

// CanonicalCatalogValue keeps only catalog keys, coerces types, and rejects
// unknown namespaces or extra keys. Secret-looking keys stay blocked.
func CanonicalCatalogValue(namespace string, value json.RawMessage) ([]byte, error) {
	category, ok := SettingCategoryByKey(namespace)
	if !ok {
		return nil, ErrSettingsInvalid
	}
	var incoming map[string]any
	if len(value) == 0 {
		incoming = map[string]any{}
	} else if err := json.Unmarshal(value, &incoming); err != nil {
		return nil, ErrSettingsInvalid
	}
	known := make(map[string]SettingField, len(category.Fields))
	for _, field := range category.Fields {
		known[field.Key] = field
	}
	for key := range incoming {
		if _, exists := known[key]; !exists {
			return nil, ErrSettingsInvalid
		}
	}
	out := make(map[string]any, len(category.Fields))
	for _, field := range category.Fields {
		typed, err := coerceSettingField(field, incoming[field.Key])
		if err != nil {
			return nil, err
		}
		out[field.Key] = typed
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, ErrSettingsInvalid
	}
	return CanonicalSettingValue(namespace, encoded)
}

func coerceSettingField(field SettingField, raw any) (any, error) {
	switch field.Type {
	case SettingFieldNumber:
		switch current := raw.(type) {
		case nil:
			return json.Number("0"), nil
		case json.Number:
			return current, nil
		case float64:
			return json.Number(strconv.FormatFloat(current, 'f', -1, 64)), nil
		case string:
			trimmed := strings.TrimSpace(current)
			if trimmed == "" {
				return json.Number("0"), nil
			}
			if _, err := strconv.ParseFloat(trimmed, 64); err != nil {
				return nil, ErrSettingsInvalid
			}
			return json.Number(trimmed), nil
		default:
			return nil, ErrSettingsInvalid
		}
	case SettingFieldBool:
		switch current := raw.(type) {
		case nil:
			return false, nil
		case bool:
			return current, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(current)) {
			case "true", "1", "yes":
				return true, nil
			case "false", "0", "no", "":
				return false, nil
			default:
				return nil, ErrSettingsInvalid
			}
		default:
			return nil, ErrSettingsInvalid
		}
	case SettingFieldSelect:
		text := strings.TrimSpace(asSettingString(raw))
		if len(field.Options) == 0 {
			return text, nil
		}
		if text == "" {
			return field.Options[0].Value, nil
		}
		for _, option := range field.Options {
			if option.Value == text {
				return text, nil
			}
		}
		return nil, ErrSettingsInvalid
	default:
		return asSettingString(raw), nil
	}
}

func asSettingString(raw any) string {
	switch current := raw.(type) {
	case nil:
		return ""
	case string:
		return current
	case json.Number:
		return current.String()
	default:
		encoded, err := json.Marshal(current)
		if err != nil {
			return ""
		}
		return strings.Trim(string(encoded), `"`)
	}
}
