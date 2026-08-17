//go:build integration

package mysql

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"testing"

	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
	"github.com/liveshop-platform/module-platform/internal/data/secretbox"
)

func TestLiveProviderVersionDefaultIdempotencyRetireAndCiphertext(t *testing.T) {
	database := openIntegrationDatabase(t)
	box, err := secretbox.New("integration-key", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	repository := NewLiveProviderRepository(database, box)
	scope := providermodel.Scope{Realm: "PLATFORM", Subject: "operator"}

	first := providerUpsert("command-1", "srs-main", 0, true)
	first.Secret = providermodel.CredentialChange{Mode: providermodel.SecretReplace, Value: "never-store-this-plaintext"}
	created, err := repository.Upsert(context.Background(), scope, first, providermodel.RequestHash(first))
	if err != nil {
		t.Fatal(err)
	}
	if created.Version != 1 || !created.Enabled || !created.IsDefault || !created.Credentials.SecretSet {
		t.Fatalf("created=%#v", created)
	}

	replayed, err := repository.Upsert(context.Background(), scope, first, providermodel.RequestHash(first))
	if err != nil || replayed.Version != 1 || replayed.ID != created.ID {
		t.Fatalf("replay=%#v err=%v", replayed, err)
	}
	changedReplay := first
	changedReplay.Name = "different"
	if _, err := repository.Upsert(context.Background(), scope, changedReplay, providermodel.RequestHash(changedReplay)); !errors.Is(err, providermodel.ErrConflict) {
		t.Fatalf("command reuse err=%v", err)
	}

	second := providerUpsert("command-2", "cloud-main", 0, true)
	cloud, err := repository.Upsert(context.Background(), scope, second, providermodel.RequestHash(second))
	if err != nil {
		t.Fatal(err)
	}
	if !cloud.IsDefault {
		t.Fatal("new default was not persisted")
	}
	items, err := repository.List(context.Background(), scope, providermodel.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	var srs providermodel.Provider
	for _, item := range items {
		if item.Code == "srs-main" {
			srs = item
		}
	}
	if srs.IsDefault || srs.Version != 2 {
		t.Fatalf("previous default did not advance immutable version: %#v", srs)
	}

	var currentLeak, historyLeak int
	for _, query := range []struct {
		target *int
		sql    string
	}{{&currentLeak, `SELECT COUNT(*) FROM live_provider WHERE CONVERT(credential_ciphertext USING utf8mb4) LIKE '%never-store-this-plaintext%'`}, {&historyLeak, `SELECT COUNT(*) FROM live_provider_version WHERE CONVERT(credential_ciphertext USING utf8mb4) LIKE '%never-store-this-plaintext%'`}} {
		if err := database.QueryRowContext(context.Background(), query.sql).Scan(query.target); err != nil {
			t.Fatal(err)
		}
	}
	if currentLeak != 0 || historyLeak != 0 {
		t.Fatal("credential plaintext reached MySQL")
	}

	retire := providermodel.Retire{CommandKey: "command-3", Code: cloud.Code, ExpectedVersion: cloud.Version}
	retired, err := repository.Retire(context.Background(), scope, retire, providermodel.RequestHash(retire))
	if err != nil {
		t.Fatal(err)
	}
	if retired.Lifecycle != providermodel.LifecycleRetired || retired.Enabled || retired.IsDefault || retired.Version != 2 {
		t.Fatalf("retired=%#v", retired)
	}
	var versions, audits, commands int
	if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM live_provider_version`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM platform_audit_event WHERE resource_type='platform.live-provider'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM live_provider_command`).Scan(&commands); err != nil {
		t.Fatal(err)
	}
	if versions != 4 || audits != 3 || commands != 3 {
		t.Fatalf("versions=%d audits=%d commands=%d", versions, audits, commands)
	}
}

func TestLiveProviderConcurrentExpectedVersionHasSingleWinner(t *testing.T) {
	database := openIntegrationDatabase(t)
	box, _ := secretbox.New("integration-key", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{8}, 32)))
	repository := NewLiveProviderRepository(database, box)
	scope := providermodel.Scope{Realm: "PLATFORM", Subject: "operator"}
	base := providerUpsert("create", "srs-main", 0, false)
	created, err := repository.Upsert(context.Background(), scope, base, providermodel.RequestHash(base))
	if err != nil {
		t.Fatal(err)
	}

	commands := []providermodel.Upsert{providerUpsert("update-a", created.Code, created.Version, false), providerUpsert("update-b", created.Code, created.Version, false)}
	commands[0].Name, commands[1].Name = "A", "B"
	var wg sync.WaitGroup
	errorsFound := make([]error, 2)
	for index := range commands {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, errorsFound[index] = repository.Upsert(context.Background(), scope, commands[index], providermodel.RequestHash(commands[index]))
		}(index)
	}
	wg.Wait()
	succeeded, conflicted := 0, 0
	for _, err := range errorsFound {
		if err == nil {
			succeeded++
		} else if errors.Is(err, providermodel.ErrConflict) {
			conflicted++
		} else {
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("success=%d conflict=%d errors=%v", succeeded, conflicted, errorsFound)
	}
	var versions, audits int
	if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM live_provider_version WHERE provider_code='srs-main'`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM platform_audit_event WHERE resource_id='srs-main'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if versions != 2 || audits != 2 {
		t.Fatalf("versions=%d audits=%d", versions, audits)
	}
}

func providerUpsert(command, code string, expected int64, isDefault bool) providermodel.Upsert {
	return providermodel.Upsert{CommandKey: command, ExpectedVersion: expected, Code: code, Name: code, Driver: providermodel.DriverSRS, App: "live", TTLSeconds: 7200, IsDefault: isDefault, Secret: providermodel.CredentialChange{Mode: providermodel.SecretKeep}, AppCertificate: providermodel.CredentialChange{Mode: providermodel.SecretClear}, CustomerCredential: providermodel.CredentialChange{Mode: providermodel.SecretClear}}
}
