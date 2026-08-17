package model

import (
	"errors"
	"testing"
)

var testScope = Scope{Realm: "PLATFORM", Subject: "operator-1"}

func TestPlatformScopeUsesGlobalCatalogueWithoutShopContext(t *testing.T) {
	if !testScope.Valid() {
		t.Fatal("platform-wide provider scope rejected an Admin session without app or merchant context")
	}
}

func TestNormalizeAndValidateProviderDerivesDriverOwnedShape(t *testing.T) {
	input := NormalizeUpsert(Upsert{
		CommandKey: "command-1", Code: " Agora-MG ", Name: " Media Gateway ", Driver: "agora_media_gateway",
		AgoraAppID: "app-id", Codec: "H264", TTLSeconds: 7200,
		CustomerCredential: CredentialChange{Mode: SecretReplace, Value: "customer-key", SecondaryValue: "customer-secret"},
		Secret:             CredentialChange{Mode: SecretClear}, AppCertificate: CredentialChange{Mode: SecretClear},
	})
	if input.Code != "agora-mg" || input.Driver != DriverAgoraMediaGateway || input.Codec != "h264" || input.Region != "na" {
		t.Fatalf("normalization mismatch: %#v", input)
	}
	if kind, ok := KindFor(input.Driver); !ok || kind != KindRTC {
		t.Fatalf("kind=%s ok=%v", kind, ok)
	}
	if err := ValidateUpsert(testScope, input); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeProviderClearsFieldsAndCredentialsOutsideSelectedDriver(t *testing.T) {
	stale := Upsert{
		CommandKey: "command-2", Code: "agora", Name: "Agora", Driver: DriverAgora,
		App: "live", PushDomain: "push.example.com", PullDomain: "pull.example.com",
		AgoraAppID: "app-id", Codec: "VP8", IngestDomain: "stale.example.com", Region: "EU",
		TTLSeconds: 7200,
		Secret:     CredentialChange{Mode: SecretKeep}, AppCertificate: CredentialChange{Mode: SecretKeep}, CustomerCredential: CredentialChange{Mode: SecretClear},
	}
	normalized := NormalizeUpsert(stale)
	if normalized.App != "" || normalized.PushDomain != "" || normalized.PullDomain != "" || normalized.Region != "" || normalized.IngestDomain != "" {
		t.Fatalf("Agora retained fields owned by another driver: %#v", normalized)
	}
	if normalized.Secret.Mode != SecretClear {
		t.Fatalf("Agora retained RTMP secret change: %#v", normalized.Secret)
	}
	if err := ValidateUpsert(testScope, normalized); err != nil {
		t.Fatal(err)
	}

	static := NormalizeUpsert(Upsert{
		CommandKey: "command-3", Code: "static", Name: "Static", Driver: DriverStatic,
		Secret: CredentialChange{Mode: SecretReplace, Value: "stale"}, AppCertificate: CredentialChange{Mode: SecretKeep}, CustomerCredential: CredentialChange{Mode: SecretKeep},
	})
	if static.App != "live" || static.Secret.Mode != SecretClear || static.AppCertificate.Mode != SecretClear || static.CustomerCredential.Mode != SecretClear {
		t.Fatalf("static driver normalization mismatch: %#v", static)
	}
	if err := ValidateUpsert(testScope, static); err != nil {
		t.Fatal(err)
	}
}

func TestProviderValidationRejectsInvalidStateAndPartialCredentialPair(t *testing.T) {
	base := NormalizeUpsert(Upsert{CommandKey: "command-1", Code: "srs", Name: "SRS", Driver: DriverSRS, TTLSeconds: 7200, Secret: CredentialChange{Mode: SecretKeep}, AppCertificate: CredentialChange{Mode: SecretClear}, CustomerCredential: CredentialChange{Mode: SecretClear}})
	for name, mutate := range map[string]func(*Upsert){
		"partial customer":   func(value *Upsert) { value.CustomerCredential = CredentialChange{Mode: SecretReplace, Value: "key"} },
		"rtc fields on rtmp": func(value *Upsert) { value.AgoraAppID = "unexpected" },
	} {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if !errors.Is(ValidateUpsert(testScope, value), ErrInvalid) {
				t.Fatal("invalid provider was accepted")
			}
		})
	}
}

func TestMaskSecretDoesNotExposeShortValues(t *testing.T) {
	if mask := MaskSecret("short"); mask != "*****" {
		t.Fatalf("short mask=%q", mask)
	}
	if mask := MaskSecret("1234567890abcdef"); mask != "123456…cdef" {
		t.Fatalf("long mask=%q", mask)
	}
}
