package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalExportDigestIsStableAndSensitive(t *testing.T) {
	first, err := json.Marshal(bundle{SchemaVersion: schemaVersion, Tables: []tableBundle{{Name: "roles", Present: true, Rows: []map[string]any{{"id": int64(1), "code": "owner"}}}, {Name: "grants", Present: false, Rows: []map[string]any{}}}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(bundle{SchemaVersion: schemaVersion, Tables: []tableBundle{{Name: "roles", Present: true, Rows: []map[string]any{{"code": "owner", "id": int64(1)}}}, {Name: "grants", Present: false, Rows: []map[string]any{}}}})
	if err != nil {
		t.Fatal(err)
	}
	if digestOf(first) != digestOf(second) {
		t.Fatal("equivalent export payload did not converge to one digest")
	}
	var changed bundle
	if err := json.Unmarshal(second, &changed); err != nil {
		t.Fatal(err)
	}
	changed.Tables[0].Rows[0]["code"] = "admin"
	changedPayload, _ := json.Marshal(changed)
	if digestOf(first) == digestOf(changedPayload) {
		t.Fatal("changed authorization facts reused the previous digest")
	}
	if err := json.Unmarshal(second, &changed); err != nil {
		t.Fatal(err)
	}
	changed.Tables[1].Present = true
	changedPayload, _ = json.Marshal(changed)
	if digestOf(first) == digestOf(changedPayload) {
		t.Fatal("present empty table and absent table reused one digest")
	}
}

func TestFinalizeRequiresExactSubscriptionImportReceipt(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	receipt := subscriptionImportReceipt{SchemaVersion: schemaVersion, Source: "liveshop-platform-authorization", ImportID: "import-1", SHA256: digest, RowCount: 20, Imported: true, ImportedAt: "2026-08-14T08:00:00Z", TargetSubscriptionInstance: "subscription-prod", TargetSubscriptionSchemaVersion: 3, TargetImportedRowCount: 4, TargetProjectionDigest: strings.Repeat("b", 64), KeyID: "subscription-receipt-1"}
	receipt.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, subscriptionReceiptSigningInput(receipt)))
	path := filepath.Join(t.TempDir(), "subscription-receipt.json")
	document, _ := json.Marshal(receipt)
	if err := os.WriteFile(path, document, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifySubscriptionReceipt(path, receipt.KeyID, base64.RawURLEncoding.EncodeToString(publicKey), receipt.TargetSubscriptionInstance, 3, digest, 20); err != nil {
		t.Fatal(err)
	}
	receipt.TargetImportedRowCount++
	document, _ = json.Marshal(receipt)
	_ = os.WriteFile(path, document, 0o600)
	if _, err := verifySubscriptionReceipt(path, receipt.KeyID, base64.RawURLEncoding.EncodeToString(publicKey), receipt.TargetSubscriptionInstance, 3, digest, 20); err == nil {
		t.Fatal("tampered target row count was accepted")
	}
}

func TestReceiptEvidenceMakesIdenticalRetryIdempotent(t *testing.T) {
	receipt := importReceipt{
		SchemaVersion: schemaVersion, Source: "liveshop-platform-authorization", ImportID: "import-1",
		SHA256: "abc", RowCount: 7, Imported: true, ImportedAt: "2026-08-14T08:00:00.123Z",
		TargetIdentityInstance: "identity-prod-cn", TargetIdentitySchemaVersion: 7, KeyID: "receipt-key-1",
	}
	first, err := evidenceFrom(receipt, "receipt-digest")
	if err != nil {
		t.Fatal(err)
	}
	second, err := evidenceFrom(receipt, "receipt-digest")
	if err != nil {
		t.Fatal(err)
	}
	if !first.same(second) {
		t.Fatal("the exact same signed receipt did not converge to one acknowledgement")
	}
	second.ImportID = "different-import"
	if first.same(second) {
		t.Fatal("a different Identity import was accepted as the acknowledged receipt")
	}
}

func TestFinalizeRequiresExactIdentityImportReceipt(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyID := "identity-authorization-import-1"
	digest := "0123456789abcdef"
	path := filepath.Join(t.TempDir(), "receipt.json")
	write := func(receipt importReceipt) {
		document, err := json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, document, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	valid := importReceipt{SchemaVersion: schemaVersion, Source: "liveshop-platform-authorization", ImportID: "import-2026-08-14", SHA256: digest, RowCount: 4, Imported: true, ImportedAt: "2026-08-14T08:00:00Z", TargetIdentityInstance: "identity-prod-cn", TargetIdentitySchemaVersion: 7, KeyID: keyID}
	valid.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, receiptSigningInput(valid)))
	write(valid)
	if _, err := verifyIdentityReceipt(path, keyID, base64.RawURLEncoding.EncodeToString(publicKey), "identity-prod-cn", 7, digest, 4); err != nil {
		t.Fatalf("valid Identity import receipt was rejected: %v", err)
	}
	resign := func(receipt importReceipt) importReceipt {
		receipt.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, receiptSigningInput(receipt)))
		return receipt
	}
	tests := map[string]func(importReceipt) importReceipt{
		"schema":          func(r importReceipt) importReceipt { r.SchemaVersion++; return r },
		"source":          func(r importReceipt) importReceipt { r.Source = "other"; return r },
		"import id":       func(r importReceipt) importReceipt { r.ImportID = ""; return r },
		"digest":          func(r importReceipt) importReceipt { r.SHA256 = "different"; return r },
		"row count":       func(r importReceipt) importReceipt { r.RowCount++; return r },
		"imported":        func(r importReceipt) importReceipt { r.Imported = false; return r },
		"import time":     func(r importReceipt) importReceipt { r.ImportedAt = "invalid"; return r },
		"target instance": func(r importReceipt) importReceipt { r.TargetIdentityInstance = "other"; return r },
		"target schema":   func(r importReceipt) importReceipt { r.TargetIdentitySchemaVersion++; return r },
		"key id":          func(r importReceipt) importReceipt { r.KeyID = "other"; return r },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			write(resign(mutate(valid)))
			if _, err := verifyIdentityReceipt(path, keyID, base64.RawURLEncoding.EncodeToString(publicKey), "identity-prod-cn", 7, digest, 4); err == nil {
				t.Fatal("semantically mismatched signed Identity import receipt was accepted")
			}
		})
	}
	tamperedSignature := valid
	tamperedSignature.Signature = base64.RawURLEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	write(tamperedSignature)
	if _, err := verifyIdentityReceipt(path, keyID, base64.RawURLEncoding.EncodeToString(publicKey), "identity-prod-cn", 7, digest, 4); err == nil {
		t.Fatal("invalid Identity receipt signature was accepted")
	}
}
