package module

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/lvtuopen-ai/kernel-go/apperror"
)

var ErrNotFound = apperror.New("platform.registry.release_not_found", "module release not found")

type release struct {
	Manifest modulemanifest.Manifest
	Digest   string
}

type ReleaseInfo struct {
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

type ModuleInfo struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	ActiveVersion string        `json:"activeVersion,omitempty"`
	Releases      []ReleaseInfo `json:"releases"`
}

// CapabilityRelease is the immutable capability contract published by one
// module release. The registry manifest remains the sole source of truth.
type CapabilityRelease struct {
	Version       string                                `json:"version"`
	Digest        string                                `json:"digest"`
	Active        bool                                  `json:"active"`
	Backend       modulemanifest.Backend                `json:"backend"`
	Permissions   []modulemanifest.PermissionDefinition `json:"permissions"`
	Contributions []modulemanifest.Contribution         `json:"contributions"`
}

type ModuleCapabilityCatalog struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	ActiveVersion string              `json:"activeVersion,omitempty"`
	Releases      []CapabilityRelease `json:"releases"`
}

type AuditActor struct {
	Realm      string
	AppID      int64
	MerchantID int64
	Subject    string
}

// Store is the registry domain model. Runtime uses its PostgreSQL-backed mode;
// NewStore keeps a deterministic in-memory instance for unit tests only.
type Store struct {
	mu       sync.RWMutex
	db       *sql.DB
	releases map[string]map[string]release
	active   map[string]string
	revision uint64
}

func NewStore() *Store {
	return &Store{releases: map[string]map[string]release{}, active: map[string]string{}, revision: 1}
}

// NewPostgresStore creates the production control-plane store. Migrations are
// applied separately so runtime workloads never mutate production schema.
func NewPostgresStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("registry database is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if _, err := store.loadSQL(ctx, nil, false); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Register(ctx context.Context, manifest modulemanifest.Manifest) (string, error) {
	if s.db != nil {
		return s.registerSQL(ctx, manifest)
	}
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	digest, err := manifest.Digest()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	versions := s.releases[manifest.Metadata.ID]
	if versions == nil {
		versions = map[string]release{}
		s.releases[manifest.Metadata.ID] = versions
	}
	if existing, ok := versions[manifest.Metadata.Version]; ok {
		if existing.Digest != digest {
			return "", errors.New("immutable module release content differs")
		}
		return digest, nil
	}
	versions[manifest.Metadata.Version] = release{Manifest: manifest, Digest: digest}
	return digest, nil
}

func (s *Store) Activate(ctx context.Context, moduleID, version string) error {
	if s.db != nil {
		return s.activateSQL(ctx, moduleID, version)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	versions := s.releases[moduleID]
	if _, ok := versions[version]; !ok {
		return ErrNotFound
	}
	if s.active[moduleID] == version {
		return nil
	}
	next := make(map[string]string, len(s.active)+1)
	for id, current := range s.active {
		next[id] = current
	}
	next[moduleID] = version
	if err := s.validateActiveRoutes(next); err != nil {
		return err
	}
	s.active = next
	s.revision++
	return nil
}

func (s *Store) Deactivate(ctx context.Context, moduleID string) error {
	if s.db != nil {
		return s.transaction(ctx, func(ctx context.Context, tx *sql.Tx, snapshot *Store) error {
			if err := snapshot.Deactivate(ctx, moduleID); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE platform_permission_catalog SET active=FALSE,updated_at=NOW() WHERE module_id=$1`, moduleID); err != nil {
				return err
			}
			return persistSQL(ctx, tx, snapshot)
		})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.active[moduleID]; !ok {
		return ErrNotFound
	}
	delete(s.active, moduleID)
	s.revision++
	return nil
}

// ActivateAudited changes the operator-visible active release and appends its
// audit event in the same serializable PostgreSQL transaction.
func (s *Store) ActivateAudited(ctx context.Context, actor AuditActor, moduleID, version string) error {
	if s.db == nil {
		return s.Activate(ctx, moduleID, version)
	}
	return s.transaction(ctx, func(ctx context.Context, tx *sql.Tx, snapshot *Store) error {
		if snapshot.active[moduleID] == version {
			return nil
		}
		if err := snapshot.Activate(ctx, moduleID, version); err != nil {
			return err
		}
		manifest := snapshot.releases[moduleID][version].Manifest
		if err := syncPermissionCatalogSQL(ctx, tx, manifest); err != nil {
			return err
		}
		if err := persistSQL(ctx, tx, snapshot); err != nil {
			return err
		}
		return auditRegistryMutation(ctx, tx, actor, "registry.module.activate", moduleID, map[string]string{"version": version})
	})
}

// DeactivateAudited removes an active module and writes the matching audit
// event atomically. Platform self-deactivation is rejected by the HTTP domain
// boundary before this method is called.
func (s *Store) DeactivateAudited(ctx context.Context, actor AuditActor, moduleID string) error {
	if s.db == nil {
		return s.Deactivate(ctx, moduleID)
	}
	return s.transaction(ctx, func(ctx context.Context, tx *sql.Tx, snapshot *Store) error {
		version := snapshot.active[moduleID]
		if err := snapshot.Deactivate(ctx, moduleID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE platform_permission_catalog SET active=FALSE,updated_at=NOW() WHERE module_id=$1`, moduleID); err != nil {
			return err
		}
		if err := persistSQL(ctx, tx, snapshot); err != nil {
			return err
		}
		return auditRegistryMutation(ctx, tx, actor, "registry.module.deactivate", moduleID, map[string]string{"version": version})
	})
}

func auditRegistryMutation(ctx context.Context, tx *sql.Tx, actor AuditActor, action, moduleID string, details any) error {
	if actor.Realm == "" || actor.AppID <= 0 || actor.MerchantID <= 0 || actor.Subject == "" {
		return errors.New("registry audit actor is required")
	}
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	random := make([]byte, 18)
	if _, err := rand.Read(random); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO platform_audit_event(event_id,realm,app_id,merchant_id,actor_subject,action,resource_type,resource_id,result,details)
		VALUES($1,$2,$3,$4,$5,$6,'platform.module',$7,'SUCCEEDED',$8)`, base64.RawURLEncoding.EncodeToString(random), actor.Realm, actor.AppID, actor.MerchantID, actor.Subject, action, moduleID, payload)
	return err
}

func (s *Store) Modules(ctx context.Context) ([]ModuleInfo, error) {
	if s.db != nil {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		snapshot, err := s.loadSQL(ctx, nil, false)
		if err != nil {
			return nil, err
		}
		return snapshot.Modules(ctx)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	output := make([]ModuleInfo, 0, len(s.releases))
	for moduleID, versions := range s.releases {
		item := ModuleInfo{ID: moduleID, ActiveVersion: s.active[moduleID]}
		if active, ok := versions[item.ActiveVersion]; ok {
			item.Name = active.Manifest.Metadata.Name
		}
		for version, current := range versions {
			item.Releases = append(item.Releases, ReleaseInfo{Version: version, Digest: current.Digest})
		}
		sort.Slice(item.Releases, func(i, j int) bool { return item.Releases[i].Version < item.Releases[j].Version })
		if item.Name == "" && len(item.Releases) > 0 {
			item.Name = versions[item.Releases[len(item.Releases)-1].Version].Manifest.Metadata.Name
		}
		output = append(output, item)
	}
	sort.Slice(output, func(i, j int) bool { return output[i].ID < output[j].ID })
	return output, nil
}

// CapabilityCatalogs returns an immutable, machine-readable view of every
// registered module release. It never probes a running module, so discovery
// cannot drift from the release that was registered and activated.
func (s *Store) CapabilityCatalogs(ctx context.Context) (uint64, []ModuleCapabilityCatalog, error) {
	if s.db != nil {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		snapshot, err := s.loadSQL(ctx, nil, false)
		if err != nil {
			return 0, nil, err
		}
		return snapshot.CapabilityCatalogs(ctx)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	output := make([]ModuleCapabilityCatalog, 0, len(s.releases))
	for moduleID, versions := range s.releases {
		item := ModuleCapabilityCatalog{ID: moduleID, ActiveVersion: s.active[moduleID]}
		if active, ok := versions[item.ActiveVersion]; ok {
			item.Name = active.Manifest.Metadata.Name
		}
		for version, current := range versions {
			spec, err := cloneSpec(current.Manifest.Spec)
			if err != nil {
				return 0, nil, err
			}
			item.Releases = append(item.Releases, CapabilityRelease{
				Version:       version,
				Digest:        current.Digest,
				Active:        s.active[moduleID] == version,
				Backend:       spec.Backend,
				Permissions:   spec.Permissions,
				Contributions: spec.Contributions,
			})
		}
		sort.Slice(item.Releases, func(i, j int) bool { return item.Releases[i].Version < item.Releases[j].Version })
		if item.Name == "" && len(item.Releases) > 0 {
			item.Name = versions[item.Releases[len(item.Releases)-1].Version].Manifest.Metadata.Name
		}
		output = append(output, item)
	}
	sort.Slice(output, func(i, j int) bool { return output[i].ID < output[j].ID })
	return s.revision, output, nil
}

func cloneSpec(source modulemanifest.Spec) (modulemanifest.Spec, error) {
	encoded, err := json.Marshal(source)
	if err != nil {
		return modulemanifest.Spec{}, err
	}
	var clone modulemanifest.Spec
	if err := json.Unmarshal(encoded, &clone); err != nil {
		return modulemanifest.Spec{}, err
	}
	return clone, nil
}

func (s *Store) validateActiveRoutes(active map[string]string) error {
	type ownedRoute struct {
		surface string
		prefix  string
		owner   string
	}
	seen := make([]ownedRoute, 0)
	for moduleID, version := range active {
		rel := s.releases[moduleID][version]
		for _, route := range rel.Manifest.Spec.Backend.HTTPRoutes {
			if route.Surface == "internal" {
				continue
			}
			prefix := normalizedRoutePrefix(route.Prefix)
			for _, current := range seen {
				if current.surface == route.Surface && current.owner != moduleID && routePrefixesOverlap(current.prefix, prefix) {
					return fmt.Errorf("route conflict %s:%s overlaps %s owned by %s", route.Surface, prefix, current.prefix, current.owner)
				}
			}
			seen = append(seen, ownedRoute{surface: route.Surface, prefix: prefix, owner: moduleID})
		}
	}
	return nil
}

func normalizedRoutePrefix(value string) string {
	value = strings.TrimRight(value, "/")
	if value == "" {
		return "/"
	}
	return value
}

func routePrefixesOverlap(left, right string) bool {
	return left == right || left == "/" || right == "/" || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

func (s *Store) Routes(ctx context.Context) (uint64, []modulemanifest.ActiveRoute, error) {
	if s.db != nil {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		snapshot, err := s.loadSQL(ctx, nil, false)
		if err != nil {
			return 0, nil, err
		}
		return snapshot.Routes(ctx)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]modulemanifest.ActiveRoute, 0)
	for moduleID, version := range s.active {
		manifest := s.releases[moduleID][version].Manifest
		for _, route := range manifest.Spec.Backend.HTTPRoutes {
			out = append(out, modulemanifest.ActiveRoute{ModuleID: moduleID, Surface: route.Surface, Prefix: strings.TrimRight(route.Prefix, "/"), Service: manifest.Spec.Backend.Service, Origin: manifest.Spec.Backend.Origin})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Prefix) == len(out[j].Prefix) {
			return out[i].ModuleID < out[j].ModuleID
		}
		return len(out[i].Prefix) > len(out[j].Prefix)
	})
	return s.revision, out, nil
}

func (s *Store) Contributions(ctx context.Context, surface string) (uint64, []modulemanifest.RuntimeContribution, error) {
	if s.db != nil {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		snapshot, err := s.loadSQL(ctx, nil, false)
		if err != nil {
			return 0, nil, err
		}
		return snapshot.Contributions(ctx, surface)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]modulemanifest.RuntimeContribution, 0)
	for moduleID, version := range s.active {
		manifest := s.releases[moduleID][version].Manifest
		for _, contribution := range manifest.Spec.Contributions {
			if contribution.Surface == surface {
				out = append(out, modulemanifest.RuntimeContribution{ModuleID: moduleID, ModuleVersion: version, Contribution: contribution})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left, right := out[i].Contribution, out[j].Contribution
		if left.Sort == right.Sort {
			return left.ID < right.ID
		}
		return left.Sort < right.Sort
	})
	return s.revision, out, nil
}

func (s *Store) Contribution(ctx context.Context, moduleID, version, contributionID string) (modulemanifest.Contribution, error) {
	if s.db != nil {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		snapshot, err := s.loadSQL(ctx, nil, false)
		if err != nil {
			return modulemanifest.Contribution{}, err
		}
		return snapshot.Contribution(ctx, moduleID, version, contributionID)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.active[moduleID] != version {
		return modulemanifest.Contribution{}, ErrNotFound
	}
	rel, ok := s.releases[moduleID][version]
	if !ok {
		return modulemanifest.Contribution{}, ErrNotFound
	}
	for _, contribution := range rel.Manifest.Spec.Contributions {
		if contribution.ID == contributionID {
			return contribution, nil
		}
	}
	return modulemanifest.Contribution{}, ErrNotFound
}

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) loadSQL(ctx context.Context, tx *sql.Tx, forUpdate bool) (*Store, error) {
	var query rowQuerier = s.db
	if tx != nil {
		query = tx
	}
	statement := `SELECT revision, releases, active FROM platform_registry_state WHERE id = 1`
	if forUpdate {
		statement += ` FOR UPDATE`
	}
	var revision uint64
	var releasesJSON, activeJSON []byte
	if err := query.QueryRowContext(ctx, statement).Scan(&revision, &releasesJSON, &activeJSON); err != nil {
		return nil, err
	}
	releases := map[string]map[string]release{}
	active := map[string]string{}
	if err := json.Unmarshal(releasesJSON, &releases); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(activeJSON, &active); err != nil {
		return nil, err
	}
	return &Store{releases: releases, active: active, revision: revision}, nil
}

func (s *Store) registerSQL(ctx context.Context, manifest modulemanifest.Manifest) (string, error) {
	var digest string
	err := s.transaction(ctx, func(ctx context.Context, tx *sql.Tx, snapshot *Store) error {
		current, err := snapshot.Register(ctx, manifest)
		if err != nil {
			return err
		}
		digest = current
		return persistSQL(ctx, tx, snapshot)
	})
	return digest, err
}

func (s *Store) activateSQL(ctx context.Context, moduleID, version string) error {
	return s.transaction(ctx, func(ctx context.Context, tx *sql.Tx, snapshot *Store) error {
		if err := snapshot.Activate(ctx, moduleID, version); err != nil {
			return err
		}
		manifest := snapshot.releases[moduleID][version].Manifest
		if err := syncPermissionCatalogSQL(ctx, tx, manifest); err != nil {
			return err
		}
		return persistSQL(ctx, tx, snapshot)
	})
}

func syncPermissionCatalogSQL(ctx context.Context, tx *sql.Tx, manifest modulemanifest.Manifest) error {
	codes := make([]string, 0, len(manifest.Spec.Permissions))
	for _, permission := range manifest.Spec.Permissions {
		codes = append(codes, permission.Code)
		result, err := tx.ExecContext(ctx, `INSERT INTO platform_permission_catalog(module_id,permission_code,name,resource_code,action,description,active,release_version,updated_at)
			VALUES($1,$2,$3,$4,$5,$6,TRUE,$7,NOW())
			ON CONFLICT(permission_code) DO UPDATE SET module_id=EXCLUDED.module_id,name=EXCLUDED.name,resource_code=EXCLUDED.resource_code,action=EXCLUDED.action,description=EXCLUDED.description,active=TRUE,release_version=EXCLUDED.release_version,updated_at=NOW()
			WHERE platform_permission_catalog.module_id=EXCLUDED.module_id`, manifest.Metadata.ID, permission.Code, permission.Name, permission.Resource, permission.Action, permission.Description, manifest.Metadata.Version)
		if err != nil {
			return err
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return fmt.Errorf("permission %s is owned by another module", permission.Code)
		}
	}
	if len(codes) == 0 {
		_, err := tx.ExecContext(ctx, `UPDATE platform_permission_catalog SET active=FALSE,updated_at=NOW() WHERE module_id=$1`, manifest.Metadata.ID)
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE platform_permission_catalog SET active=FALSE,updated_at=NOW() WHERE module_id=$1 AND NOT(permission_code = ANY($2))`, manifest.Metadata.ID, codes)
	return err
}

func (s *Store) transaction(ctx context.Context, operation func(context.Context, *sql.Tx, *Store) error) error {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		last = s.transactionOnce(ctx, operation)
		if !retryableTransactionError(last) {
			return last
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return last
}

func (s *Store) transactionOnce(parent context.Context, operation func(context.Context, *sql.Tx, *Store) error) error {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	snapshot, err := s.loadSQL(ctx, tx, true)
	if err != nil {
		return err
	}
	if err := operation(ctx, tx, snapshot); err != nil {
		return err
	}
	return tx.Commit()
}

func retryableTransactionError(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && (postgresError.Code == "40001" || postgresError.Code == "40P01")
}

func persistSQL(ctx context.Context, tx *sql.Tx, snapshot *Store) error {
	releases, err := json.Marshal(snapshot.releases)
	if err != nil {
		return err
	}
	active, err := json.Marshal(snapshot.active)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE platform_registry_state SET revision = $1, releases = $2, active = $3, updated_at = NOW() WHERE id = 1`, snapshot.revision, releases, active)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return errors.New("registry state update did not affect exactly one row")
	}
	return nil
}
