package model

import "testing"

func TestValidatePhoneAndRoute(t *testing.T) {
	if ValidatePhone("13800138000") || !ValidatePhone("+8613800138000") {
		t.Fatal("E.164 phone rule drifted")
	}
	channels := []Channel{
		{Code: "global", Driver: DriverMock, Region: "*", Priority: 10, Enabled: true, Lifecycle: LifecycleActive},
		{Code: "cn", Driver: DriverAliyun, Region: "+86", Priority: 1, Enabled: true, Lifecycle: LifecycleActive},
		{Code: "off", Driver: DriverMock, Region: "+86", Priority: 99, Enabled: false, Lifecycle: LifecycleActive},
	}
	routed := RouteChannels("+8613800138000", channels)
	if len(routed) != 2 || routed[0].Code != "cn" || routed[1].Code != "global" {
		t.Fatalf("route=%#v", routed)
	}
}

func TestSecretMergeAndDriverSchema(t *testing.T) {
	if _, ok := DefinitionFor(DriverAliyun); !ok {
		t.Fatal("aliyun driver missing")
	}
	merged := ApplySecrets(map[string]string{"access_key_secret": "old", "other": "x"}, map[string]CredentialChange{
		"access_key_secret": {Mode: SecretKeep},
		"api_key":           {Mode: SecretReplace, Value: "new"},
	})
	if merged["access_key_secret"] != "old" || merged["api_key"] != "new" {
		t.Fatalf("merged=%#v", merged)
	}
	cleared := ApplySecrets(merged, map[string]CredentialChange{"access_key_secret": {Mode: SecretClear}})
	if _, ok := cleared["access_key_secret"]; ok {
		t.Fatal("secret was not cleared")
	}
	if MaskSecret("abcdefghijklmn") == "abcdefghijklmn" || MaskSecret("") != "" {
		t.Fatal("mask leaked or empty mask drifted")
	}
}

func TestUpsertChannelRejectsUnknownSecret(t *testing.T) {
	input := NormalizeUpsertChannel(UpsertChannel{
		CommandKey: "cmd-1", Code: "aliyun-cn", Name: "中国通道", Driver: DriverAliyun, Region: "+86",
		PublicConfig: map[string]string{"access_key_id": "id", "sign_name": "签名", "template_code": "SMS_1"},
		Secrets:      map[string]CredentialChange{"access_key_secret": {Mode: SecretReplace, Value: "secret"}, "unknown": {Mode: SecretReplace, Value: "x"}},
	})
	if err := ValidateUpsertChannel(Scope{Realm: "PLATFORM", Subject: "op"}, input); err != ErrInvalid {
		t.Fatalf("unknown secret was accepted: %v", err)
	}
}

func TestGrantEmptyMeansUnrestricted(t *testing.T) {
	input := PutMerchantGrant{CommandKey: "g1", MerchantID: 1001, ShopID: 2001}
	if err := ValidatePutGrant(Scope{Realm: "PLATFORM", Subject: "op"}, input); err != nil {
		t.Fatal(err)
	}
	if GrantResourceID(1001, 2001) != "1001:2001" {
		t.Fatal(GrantResourceID(1001, 2001))
	}
}
