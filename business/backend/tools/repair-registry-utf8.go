//go:build ignore

// Repair local registry release JSON after a non-UTF-8 PowerShell register
// wrote contribution titles as "????". Re-reads module.json as UTF-8 and
// patches human-readable Chinese fields onto the already-registered release.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

func main() {
	dsn := flag.String("dsn", "liveshop:liveshop-local@tcp(127.0.0.1:33069)/liveshop_registry?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_0900_ai_ci", "MySQL DSN")
	manifestPath := flag.String("manifest", "", "path to module.json")
	moduleID := flag.String("module", "platform", "module id")
	version := flag.String("version", "2.0.1", "registered version to repair")
	flag.Parse()
	if *manifestPath == "" {
		fatal("manifest is required")
	}

	raw, err := os.ReadFile(*manifestPath)
	if err != nil {
		fatal(err)
	}
	var source modulemanifest.Manifest
	if err := json.Unmarshal(raw, &source); err != nil {
		fatal(err)
	}

	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	var releasesJSON, activeJSON []byte
	var revision uint64
	if err := db.QueryRow(`SELECT revision, releases, active FROM platform_registry_state WHERE id = 1`).Scan(&revision, &releasesJSON, &activeJSON); err != nil {
		fatal(err)
	}
	state := &model.RegistryState{
		Revision: revision,
		Releases: map[string]map[string]model.Release{},
		Active:   map[string]string{},
	}
	if err := json.Unmarshal(releasesJSON, &state.Releases); err != nil {
		fatal(err)
	}
	if err := json.Unmarshal(activeJSON, &state.Active); err != nil {
		fatal(err)
	}

	versions := state.Releases[*moduleID]
	if versions == nil {
		fatal("module not registered: " + *moduleID)
	}
	current, ok := versions[*version]
	if !ok {
		fatal("version not registered: " + *version)
	}
	patchReadableText(&current.Manifest, source)
	versions[*version] = current

	nextReleases, err := json.Marshal(state.Releases)
	if err != nil {
		fatal(err)
	}
	if _, err := db.Exec(`UPDATE platform_registry_state SET revision = revision + 1, releases = ?, updated_at = NOW(3) WHERE id = 1`, nextReleases); err != nil {
		fatal(err)
	}
	fmt.Printf("repaired readable UTF-8 text for %s@%s (registry revision bumped)\n", *moduleID, *version)
}

func patchReadableText(dst *modulemanifest.Manifest, src modulemanifest.Manifest) {
	dst.Metadata.Name = src.Metadata.Name
	byContribution := map[string]modulemanifest.Contribution{}
	for _, item := range src.Spec.Contributions {
		byContribution[item.ID] = item
	}
	for i := range dst.Spec.Contributions {
		good, ok := byContribution[dst.Spec.Contributions[i].ID]
		if !ok {
			continue
		}
		dst.Spec.Contributions[i].Title = good.Title
		dst.Spec.Contributions[i].Description = good.Description
		dst.Spec.Contributions[i].Frontend.Actions = good.Frontend.Actions
		dst.Spec.Contributions[i].Frontend.Events = good.Frontend.Events
	}
	byPermission := map[string]modulemanifest.PermissionDefinition{}
	for _, item := range src.Spec.Permissions {
		byPermission[item.Code] = item
	}
	for i := range dst.Spec.Permissions {
		good, ok := byPermission[dst.Spec.Permissions[i].Code]
		if !ok {
			continue
		}
		dst.Spec.Permissions[i].Name = good.Name
		dst.Spec.Permissions[i].Description = good.Description
	}
	for i := range dst.Spec.Backend.HTTPRoutes {
		if i >= len(src.Spec.Backend.HTTPRoutes) {
			break
		}
		for j := range dst.Spec.Backend.HTTPRoutes[i].Operations {
			if j >= len(src.Spec.Backend.HTTPRoutes[i].Operations) {
				break
			}
			dst.Spec.Backend.HTTPRoutes[i].Operations[j].Summary = src.Spec.Backend.HTTPRoutes[i].Operations[j].Summary
			dst.Spec.Backend.HTTPRoutes[i].Operations[j].Description = src.Spec.Backend.HTTPRoutes[i].Operations[j].Description
		}
	}
}

func fatal(err any) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
