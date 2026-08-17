package model

import "testing"

func TestDriverDefinitionsOwnKindPushTransportAndFormSchema(t *testing.T) {
	definitions := DriverDefinitions()
	if len(definitions) != 5 {
		t.Fatalf("definitions=%d", len(definitions))
	}
	seen := map[Driver]bool{}
	for _, definition := range definitions {
		if seen[definition.Code] || definition.Name == "" || definition.Description == "" || len(definition.Fields) == 0 {
			t.Fatalf("invalid definition=%#v", definition)
		}
		seen[definition.Code] = true
		kind, found := KindFor(definition.Code)
		if !found || kind != definition.Kind {
			t.Fatalf("driver=%s kind=%s found=%v", definition.Code, kind, found)
		}
	}
	assertDriverFieldOptions(t, DriverAgora, "codec", []string{"vp8", "h264", "vp9", "av1", "h265"})
	assertDriverFieldOptions(t, DriverAgoraMediaGateway, "codec", []string{"h264", "h265"})
	if field, found := driverField(DriverStatic, "secret"); found || field.Key != "" {
		t.Fatal("static driver unexpectedly exposes a signing secret")
	}
	customerKey, found := driverField(DriverAgoraMediaGateway, "customerKey")
	if !found || !customerKey.Required || customerKey.Credential != CredentialCustomerCredential {
		t.Fatalf("media gateway customer key=%#v found=%v", customerKey, found)
	}
}

func assertDriverFieldOptions(t *testing.T, driver Driver, key string, want []string) {
	t.Helper()
	field, found := driverField(driver, key)
	if !found || len(field.Options) != len(want) {
		t.Fatalf("driver=%s field=%s options=%#v found=%v", driver, key, field.Options, found)
	}
	for index := range want {
		if field.Options[index].Value != want[index] {
			t.Fatalf("driver=%s field=%s option[%d]=%s", driver, key, index, field.Options[index].Value)
		}
	}
}

func driverField(driver Driver, key string) (ConfigField, bool) {
	definition, found := DefinitionFor(driver)
	if !found {
		return ConfigField{}, false
	}
	for _, field := range definition.Fields {
		if field.Key == key {
			return field, true
		}
	}
	return ConfigField{}, false
}
