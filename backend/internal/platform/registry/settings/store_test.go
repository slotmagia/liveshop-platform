package settings

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestValidateValueRejectsSecretMaterial(t *testing.T) {
	for _, value := range []string{`{"password":"x"}`, `{"nested":{"privateKey":"x"}}`, `{"items":[{"api_token":"x"}]}`, `[]`, `"value"`, `{"ok":true}{"trailing":true}`} {
		if _, err := validateValue("branding", json.RawMessage(value)); err == nil {
			t.Fatalf("expected rejection for %s", value)
		}
	}
	if _, err := validateValue("branding", json.RawMessage(`{"name":"LiveShop","locale":"zh-CN"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestScopeAndHelpers(t *testing.T) {
	if !(Scope{Realm: "PLATFORM", AppID: 1, MerchantID: 2, Subject: "operator"}).Valid() {
		t.Fatal("valid platform scope was rejected")
	}
	if (Scope{Realm: "UNKNOWN", AppID: 1, MerchantID: 2, Subject: "operator"}).Valid() {
		t.Fatal("unknown realm was accepted")
	}
	if _, err := New(nil); err == nil {
		t.Fatal("nil database was accepted")
	}
	if got := compact([]byte("not-json")); !bytes.Equal(got, []byte("not-json")) {
		t.Fatalf("invalid JSON compact changed bytes: %q", got)
	}
	first, second := eventID(), eventID()
	if first == second || len(first) != 24 || len(second) != 24 {
		t.Fatalf("event IDs are not unique 144-bit identifiers: %q %q", first, second)
	}
}
