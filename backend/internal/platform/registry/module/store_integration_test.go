//go:build integration

package module

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresPermissionLifecycleAndContextCancellation(t *testing.T) {
	databaseURL := os.Getenv("PLATFORM_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("PLATFORM_TEST_DATABASE_URL is not set")
	}

	base, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = base.Close() })

	schema := fmt.Sprintf("registry_it_%d", time.Now().UnixNano())
	if _, err := base.ExecContext(context.Background(), `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, cleanupErr := base.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`); cleanupErr != nil {
			t.Errorf("drop integration schema: %v", cleanupErr)
		}
	})

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	database, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	applyIntegrationMigrations(t, database)
	store, err := NewPostgresStore(database)
	if err != nil {
		t.Fatal(err)
	}
	manifest := fixture("integration-catalog", "/admin/integration-catalog")
	if _, err := store.Register(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), manifest.Metadata.ID, manifest.Metadata.Version); err != nil {
		t.Fatal(err)
	}

	permission := manifest.Spec.Permissions[0].Code
	var active bool
	var releaseVersion string
	if err := database.QueryRowContext(context.Background(), `SELECT active,release_version FROM platform_permission_catalog WHERE permission_code=$1`, permission).Scan(&active, &releaseVersion); err != nil {
		t.Fatal(err)
	}
	if !active || releaseVersion != manifest.Metadata.Version {
		t.Fatalf("activated permission lifecycle mismatch: active=%v release=%q", active, releaseVersion)
	}

	if err := store.Deactivate(context.Background(), manifest.Metadata.ID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(context.Background(), `SELECT active FROM platform_permission_catalog WHERE permission_code=$1`, permission).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active {
		t.Fatal("permission remained active after module deactivation")
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Modules(canceled); err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("caller cancellation was not propagated: %v", err)
	}
}

func applyIntegrationMigrations(t *testing.T, database *sql.DB) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "..", "migrations", "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("discover migrations: files=%d err=%v", len(files), err)
	}
	for _, path := range files {
		migration, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, execErr := database.ExecContext(context.Background(), string(migration)); execErr != nil {
			t.Fatalf("apply %s: %v", filepath.Base(path), execErr)
		}
	}
}
