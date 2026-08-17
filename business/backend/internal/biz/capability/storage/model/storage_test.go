package model

import "testing"

func TestValidateUpsertRejectsUnknownSecret(t *testing.T) {
	err := ValidateUpsert(Scope{Realm: "admin", Subject: "op"}, UpsertChannel{
		CommandKey: "cmd-1", Code: "oss-cn", Name: "杭州 OSS", Driver: DriverAliyunOSS,
		PublicConfig: map[string]string{"endpoint": "oss-cn-hangzhou.aliyuncs.com", "bucket": "demo", "access_key_id": "id"},
		Secrets:      map[string]CredentialChange{"access_key_secret": {Mode: SecretReplace, Value: "secret"}, "extra": {Mode: SecretReplace, Value: "x"}},
	})
	if err != ErrInvalid {
		t.Fatalf("got %v", err)
	}
}

func TestValidateUpsertAllowsLocalWithoutSecrets(t *testing.T) {
	err := ValidateUpsert(Scope{Realm: "admin", Subject: "op"}, UpsertChannel{
		CommandKey: "cmd-1", Code: "local", Name: "本地磁盘", Driver: DriverLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDriverDefinitionsMatchLegacyKeys(t *testing.T) {
	for _, code := range []Driver{DriverLocal, DriverAliyunOSS, DriverCloudflareR2} {
		if _, ok := DefinitionFor(code); !ok {
			t.Fatalf("missing %s", code)
		}
	}
	if keys := SecretFieldKeys(DriverAliyunOSS); len(keys) != 1 || keys[0] != "access_key_secret" {
		t.Fatalf("oss secrets=%v", keys)
	}
}
