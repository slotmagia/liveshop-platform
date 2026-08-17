//go:build integration

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

func TestMySQLPermissionLifecycleAndContextCancellation(t *testing.T) {
	database := openIntegrationDatabase(t)
	repository, err := NewReleaseRepository(context.Background(), database)
	if err != nil {
		t.Fatal(err)
	}
	manifest := integrationFixture("integration-catalog", "/admin/integration-catalog")
	if _, err := repository.Register(context.Background(), manifest); err != nil {
		t.Fatal(err)
	}
	if err := repository.Activate(context.Background(), nil, manifest.Metadata.ID, manifest.Metadata.Version); err != nil {
		t.Fatal(err)
	}

	permission := manifest.Spec.Permissions[0].Code
	var active bool
	var releaseVersion string
	if err := database.QueryRowContext(context.Background(), `SELECT active,release_version FROM platform_permission_catalog WHERE permission_code=?`, permission).Scan(&active, &releaseVersion); err != nil {
		t.Fatal(err)
	}
	if !active || releaseVersion != manifest.Metadata.Version {
		t.Fatalf("activated permission lifecycle mismatch: active=%v release=%q", active, releaseVersion)
	}

	if err := repository.Deactivate(context.Background(), nil, manifest.Metadata.ID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(context.Background(), `SELECT active FROM platform_permission_catalog WHERE permission_code=?`, permission).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active {
		t.Fatal("permission remained active after module deactivation")
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repository.Snapshot(canceled); err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("caller cancellation was not propagated: %v", err)
	}
}

func TestMySQLNavigationGroupConflictRollsBackEverySideEffect(t *testing.T) {
	database := openIntegrationDatabase(t)
	repository, err := NewReleaseRepository(context.Background(), database)
	if err != nil {
		t.Fatal(err)
	}

	first := integrationWithNavigation(
		integrationFixture("integration-catalog", "/admin/integration-catalog"),
		"admin", "commerce", "Commerce", 20,
	)
	conflicting := integrationWithNavigation(
		integrationFixture("integration-order", "/admin/integration-order"),
		"admin", "commerce", "Sales", 20,
	)
	actor := &model.RegistryAuditActor{Realm: "PLATFORM", MerchantID: 0, Subject: "integration-operator"}

	for _, manifest := range []modulemanifest.Manifest{first, conflicting} {
		if _, err := repository.Register(context.Background(), manifest); err != nil {
			t.Fatal(err)
		}
	}
	if err := repository.Activate(context.Background(), actor, first.Metadata.ID, first.Metadata.Version); err != nil {
		t.Fatal(err)
	}
	before, err := repository.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		err := repository.Activate(context.Background(), actor, conflicting.Metadata.ID, conflicting.Metadata.Version)
		if !errors.Is(err, model.ErrNavigationGroupConflict) {
			t.Fatalf("attempt %d: expected navigation group conflict, got %v", attempt, err)
		}
		assertNavigationConflictHasNoSideEffects(t, database, repository, before, first, conflicting)
	}
}

func openIntegrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("PLATFORM_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("PLATFORM_TEST_DATABASE_URL is not set")
	}

	base, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = base.Close() })

	databaseName := fmt.Sprintf("registry_it_%d", time.Now().UnixNano())
	if _, err := base.ExecContext(context.Background(), `CREATE DATABASE `+databaseName); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, cleanupErr := base.ExecContext(context.Background(), `DROP DATABASE IF EXISTS `+databaseName); cleanupErr != nil {
			t.Errorf("drop integration database: %v", cleanupErr)
		}
	})

	database, err := sql.Open("mysql", rewriteMySQLDatabase(dsn, databaseName))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	applyIntegrationMigrations(t, database)
	return database
}

func assertNavigationConflictHasNoSideEffects(
	t *testing.T,
	database *sql.DB,
	repository *ReleaseRepository,
	before *model.RegistryState,
	first modulemanifest.Manifest,
	conflicting modulemanifest.Manifest,
) {
	t.Helper()
	after, err := repository.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != before.Revision || after.ActiveVersion(first.Metadata.ID) != first.Metadata.Version || after.ActiveVersion(conflicting.Metadata.ID) != "" {
		t.Fatalf("failed activation changed registry state: before_revision=%d after_revision=%d active=%#v", before.Revision, after.Revision, after.Active)
	}

	var permissionRows int
	if err := database.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM platform_permission_catalog WHERE module_id=? OR permission_code=?`,
		conflicting.Metadata.ID, conflicting.Spec.Permissions[0].Code,
	).Scan(&permissionRows); err != nil {
		t.Fatal(err)
	}
	if permissionRows != 0 {
		t.Fatalf("failed activation wrote %d conflicting permission catalog rows", permissionRows)
	}

	var activationAudits int
	if err := database.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM platform_audit_event WHERE action='registry.module.activate' AND resource_id=?`,
		conflicting.Metadata.ID,
	).Scan(&activationAudits); err != nil {
		t.Fatal(err)
	}
	if activationAudits != 0 {
		t.Fatalf("failed activation wrote %d activation audits", activationAudits)
	}
}

// rewriteMySQLDatabase swaps the schema segment of a go-sql-driver DSN so each
// integration case owns an isolated database without parsing query options.
func rewriteMySQLDatabase(dsn, databaseName string) string {
	at := strings.Index(dsn, "@")
	slash := strings.Index(dsn[at+1:], "/")
	if at < 0 || slash < 0 {
		return dsn
	}
	slash += at + 1
	rest := dsn[slash+1:]
	if q := strings.Index(rest, "?"); q >= 0 {
		return dsn[:slash+1] + databaseName + rest[q:]
	}
	return dsn[:slash+1] + databaseName
}

func integrationFixture(id, prefix string) modulemanifest.Manifest {
	permission := id + ".item.read"
	return modulemanifest.Manifest{
		APIVersion: modulemanifest.APIVersion,
		Kind:       modulemanifest.KindModuleRelease,
		Metadata:   modulemanifest.Metadata{ID: id, Name: id, Version: "1.0.0"},
		Spec: modulemanifest.Spec{
			Backend: modulemanifest.Backend{Service: id, Origin: "http://" + id, HTTPRoutes: []modulemanifest.HTTPRoute{{
				Surface: "admin", Prefix: prefix, Operations: []modulemanifest.HTTPOperation{{
					ID: id + ".item.list", Method: "GET", Path: prefix, Summary: "List items", Description: "Lists module items.", Authentication: "module-session", Idempotency: "safe", RequiredPermissions: []string{permission},
					Responses: []modulemanifest.CapabilityResponse{{Status: 200, Description: "Item list"}},
				}},
			}}},
			Permissions: []modulemanifest.PermissionDefinition{{Code: permission, Name: "Read items", Resource: id + ".item", Action: "read"}},
		},
	}
}

func integrationWithNavigation(manifest modulemanifest.Manifest, surface, groupID, groupTitle string, groupSort int) modulemanifest.Manifest {
	permission := manifest.Spec.Permissions[0].Code
	manifest.Spec.Backend.HTTPRoutes[0].Surface = surface
	manifest.Spec.Contributions = []modulemanifest.Contribution{{
		ID:          manifest.Metadata.ID + ".page",
		Surface:     surface,
		Kind:        "page",
		Route:       "/" + manifest.Metadata.ID,
		Title:       manifest.Metadata.Name,
		Description: manifest.Metadata.Name + " integration page.",
		Navigation:  &modulemanifest.Navigation{GroupID: groupID, GroupTitle: groupTitle, GroupSort: groupSort},
		RequiredPermissions: []string{
			permission,
		},
		AllowedRoutes: []modulemanifest.AllowedRoute{{
			Methods:             []string{"GET"},
			Prefix:              manifest.Spec.Backend.HTTPRoutes[0].Prefix,
			RequiredPermissions: []string{permission},
		}},
		Artifact: modulemanifest.Artifact{
			Type:      "iframe",
			Name:      manifest.Metadata.ID + "-admin",
			Version:   manifest.Metadata.Version,
			Entry:     "https://" + manifest.Metadata.ID + ".invalid",
			Integrity: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		Frontend: modulemanifest.FrontendContract{Component: manifest.Metadata.Name + "Page"},
	}}
	return manifest
}

func applyIntegrationMigrations(t *testing.T, database *sql.DB) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "migrations", "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("discover migrations: files=%d err=%v", len(files), err)
	}
	for _, path := range files {
		migration, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		// The MySQL driver rejects multi-statement Exec unless the DSN opts in,
		// so apply one statement at a time instead of widening the test DSN.
		for _, statement := range splitSQLStatements(string(migration)) {
			if _, execErr := database.ExecContext(context.Background(), statement); execErr != nil {
				t.Fatalf("apply %s: %v\nstatement: %s", filepath.Base(path), execErr, statement)
			}
		}
	}
}

func splitSQLStatements(script string) []string {
	var statements []string
	var current strings.Builder
	delimiter := ";"
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		if fields := strings.Fields(trimmed); len(fields) == 2 && strings.EqualFold(fields[0], "DELIMITER") {
			delimiter = fields[1]
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
		if strings.HasSuffix(trimmed, delimiter) {
			statement := strings.TrimSpace(current.String())
			statement = strings.TrimSuffix(statement, delimiter)
			statement = strings.TrimSpace(statement)
			if statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
		}
	}
	if tail := strings.TrimSpace(current.String()); tail != "" {
		statements = append(statements, strings.TrimSuffix(tail, delimiter))
	}
	return statements
}
