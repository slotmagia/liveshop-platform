package migrations_test

import (
	"os"
	"strings"
	"testing"
)

func TestDropAppCommercialMakesCataloguesGlobalAndGrantsShopScoped(t *testing.T) {
	raw, err := os.ReadFile("012_drop_app_commercial.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, required := range []string{
		"SIGNAL SQLSTATE '45000'",
		"platform_drop_scope_assert_zero",
		"platform_sms_grant_rekey_guard",
		"UNIQUE KEY uk_sms_grant_shop (merchant_id, shop_id)",
		"PRIMARY KEY (merchant_id, shop_id, version)",
		"UNIQUE KEY `uk_sms_region_code` (`code`)",
		"UNIQUE KEY `uk_sms_channel_code` (`code`)",
		"UNIQUE KEY `uk_storage_channel_code` (`code`)",
		"UNIQUE KEY `uk_live_provider_code` (`code`)",
		"uk_email_config_singleton",
		"PRIMARY KEY (command_key)",
		"PRIMARY KEY (realm, merchant_id, namespace)",
		"ck_setting_scope",
		"idx_platform_audit_scope_time (realm, merchant_id, occurred_at DESC)",
		"identity_shop",
		"INSERT INTO sms_catalogue(id, revision) VALUES (1, 0)",
		"INSERT INTO storage_catalogue(id, revision) VALUES (1, 0)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("012 migration is missing contract %q", required)
		}
	}
	for _, table := range []string{
		"sms_catalogue", "sms_region", "sms_channel", "sms_command",
		"email_catalogue", "email_config", "email_command",
		"storage_catalogue", "storage_channel", "storage_command",
		"live_provider_catalogue", "live_provider", "live_provider_command",
		"platform_setting", "platform_audit_event",
	} {
		if !strings.Contains(sql, "platform_drop_scope_drop_column('"+table+"','app_id')") {
			t.Errorf("012 migration does not drop app_id from %s", table)
		}
	}
	if !strings.Contains(sql, "platform_drop_scope_drop_column('sms_merchant_grant','grant_app_id')") {
		t.Fatal("012 migration does not drop grant_app_id")
	}
	if !strings.Contains(sql, "platform_drop_scope_drop_column('sms_merchant_grant','commercial_id')") {
		t.Fatal("012 migration does not drop commercial_id")
	}
}

func TestDropAppCommercialDoesNotEditHistoricalMigrations(t *testing.T) {
	raw, err := os.ReadFile("012_drop_app_commercial.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(raw)
	for _, forbidden := range []string{
		"001_baseline",
		"007_live_provider",
		"009_sms",
		"010_email",
		"011_storage",
		"commercial_id = merchant_id",
		"commercial_id=merchant_id",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("012 migration contains forbidden text %q", forbidden)
		}
	}
}
