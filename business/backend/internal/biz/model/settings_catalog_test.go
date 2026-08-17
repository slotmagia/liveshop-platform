package model

import (
	"encoding/json"
	"testing"
)

func TestSettingCatalogOwnsOnlyPlatformFacts(t *testing.T) {
	seen := map[string]bool{}
	for _, group := range SettingCatalog() {
		if group.Key == "" || group.Label == "" || len(group.Categories) == 0 {
			t.Fatalf("invalid group %#v", group)
		}
		for _, category := range group.Categories {
			if !ValidNamespace(category.Key) || seen[category.Key] || len(category.Fields) == 0 {
				t.Fatalf("invalid category %#v", category)
			}
			seen[category.Key] = true
			for _, field := range category.Fields {
				if field.Key == "" || field.Label == "" || settingSecretKeyPattern.MatchString(field.Key) {
					t.Fatalf("invalid field %#v", field)
				}
			}
		}
	}
	if _, ok := SettingCategoryByKey("live-basic"); ok {
		t.Fatal("live settings must stay in the Live module")
	}
	if _, ok := SettingCategoryByKey("order"); ok {
		t.Fatal("trade settings must stay in the Trade module")
	}
}

func TestCanonicalCatalogValueCoercesAndRejectsExtras(t *testing.T) {
	encoded, err := CanonicalCatalogValue("site", json.RawMessage(`{"site_name":"LiveShop","customer_phone":"400","icp":""}`))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil || got["site_name"] != "LiveShop" {
		t.Fatalf("got=%s err=%v", encoded, err)
	}
	if _, err := CanonicalCatalogValue("site", json.RawMessage(`{"site_name":"x","unknown":"y"}`)); err != ErrSettingsInvalid {
		t.Fatalf("extra key: %v", err)
	}
	if _, err := CanonicalCatalogValue("branding", json.RawMessage(`{"name":"x"}`)); err != ErrSettingsInvalid {
		t.Fatalf("unknown namespace: %v", err)
	}
	encoded, err = CanonicalCatalogValue("domain-base", json.RawMessage(`{"force_https":"true","root_domain":"wopays.com"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &got); err != nil || got["force_https"] != true {
		t.Fatalf("bool coerce got=%s err=%v", encoded, err)
	}
}
