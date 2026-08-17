package model

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCanonicalSettingValueRejectsSecretMaterial(t *testing.T) {
	for _, value := range []string{`{"password":"x"}`, `{"nested":{"privateKey":"x"}}`, `{"items":[{"api_token":"x"}]}`, `[]`, `"value"`, `{"ok":true}{"trailing":true}`} {
		if _, err := CanonicalSettingValue("branding", json.RawMessage(value)); err == nil {
			t.Fatalf("expected rejection for %s", value)
		}
	}
	if _, err := CanonicalSettingValue("branding", json.RawMessage(`{"name":"LiveShop","locale":"zh-CN"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := CanonicalSettingValue("secrets", json.RawMessage(`{"name":"LiveShop"}`)); err == nil {
		t.Fatal("the reserved secrets namespace was accepted")
	}
}

func TestSettingScopeAndHelpers(t *testing.T) {
	if !(SettingScope{Realm: "PLATFORM", Subject: "operator"}).Valid() {
		t.Fatal("platform operator zero-tenant scope was rejected")
	}
	if (SettingScope{Realm: "PLATFORM", MerchantID: 2, Subject: "operator"}).Valid() {
		t.Fatal("platform scope with a merchant tenant pair was accepted")
	}
	if !(SettingScope{Realm: "MERCHANT", MerchantID: 2, Subject: "owner"}).Valid() {
		t.Fatal("merchant tenant scope was rejected")
	}
	if (SettingScope{Realm: "MERCHANT", Subject: "owner"}).Valid() {
		t.Fatal("merchant scope without a tenant pair was accepted")
	}
	if (SettingScope{Realm: "CUSTOMER", Subject: "buyer"}).Valid() {
		t.Fatal("customer realm was accepted")
	}
	if (SettingScope{Realm: "UNKNOWN", Subject: "operator"}).Valid() {
		t.Fatal("unknown realm was accepted")
	}
	if got := CompactJSON([]byte("not-json")); !bytes.Equal(got, []byte("not-json")) {
		t.Fatalf("invalid JSON compact changed bytes: %q", got)
	}
	if !ValidNamespace("branding") || ValidNamespace("Branding") || ValidNamespace("a") {
		t.Fatal("namespace pattern does not match the documented contract")
	}
}
