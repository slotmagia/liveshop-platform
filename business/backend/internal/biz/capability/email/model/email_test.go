package model

import "testing"

func TestValidateEmail(t *testing.T) {
	if !ValidateEmail("someone@example.com") || ValidateEmail("not-an-email") || ValidateEmail("a@b") {
		t.Fatal("email validation rejected a valid address or accepted an invalid one")
	}
}

func TestValidateUpsertRejectsUnknownSecret(t *testing.T) {
	err := ValidateUpsert(Scope{Realm: "admin", Subject: "op"}, UpsertConfig{
		CommandKey: "cmd-1", Driver: DriverSMTP,
		PublicConfig: map[string]string{"host": "smtp.example.com", "port": "465", "encryption": "ssl", "username": "notify@example.com"},
		Secrets:      map[string]CredentialChange{"password": {Mode: SecretReplace, Value: "secret"}, "extra": {Mode: SecretReplace, Value: "x"}},
	})
	if err != ErrInvalid {
		t.Fatalf("got %v", err)
	}
}

func TestApplySecretsKeepsUnmentioned(t *testing.T) {
	next := ApplySecrets(map[string]string{"password": "old"}, map[string]CredentialChange{"password": {Mode: SecretKeep}})
	if next["password"] != "old" {
		t.Fatalf("got %#v", next)
	}
}

func TestDriverDefinitionsIncludeSMTPSchema(t *testing.T) {
	definition, ok := DefinitionFor(DriverSMTP)
	if !ok || len(SecretFieldKeys(DriverSMTP)) != 1 {
		t.Fatalf("smtp definition=%#v ok=%v", definition, ok)
	}
}
