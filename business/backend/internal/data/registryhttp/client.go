// Package registryhttp is Platform's workload client for the independent
// Registry process. Registry owns the durable aggregate; this adapter never
// writes platform_registry_state.
package registryhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/liveshop-platform/contracts/modulemanifest"
	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/config"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
)

const (
	registryTimeout          = 10 * time.Second
	workloadTokenTTL         = time.Minute
	platformInternalAudience = "liveshop-platform-internal"
	registryPrefix           = "/internal/v1/module-registry"
)

type Client struct {
	origin string
	issuer *workloadidentity.Issuer
	http   *http.Client
}

var _ biz.ReleaseRepository = (*Client)(nil)

func New(cfg *config.Config) (*Client, error) {
	issuer, err := workloadidentity.NewIssuer(
		cfg.Registry.Workload.PrivateKey,
		cfg.Registry.Workload.KeyID,
		cfg.Registry.Workload.Issuer,
		cfg.Registry.Workload.Subject,
		platformInternalAudience,
	)
	if err != nil {
		return nil, fmt.Errorf("platform: registry workload identity: %w", err)
	}
	return &Client{
		origin: strings.TrimRight(cfg.Registry.OriginURL, "/"),
		issuer: issuer,
		http:   &http.Client{Timeout: registryTimeout},
	}, nil
}

func (c *Client) Snapshot(ctx context.Context) (*model.RegistryState, error) {
	var payload struct {
		Revision uint64                          `json:"revision"`
		Items    []model.ModuleCapabilityCatalog `json:"items"`
	}
	if err := c.get(ctx, "/capabilities", &payload); err != nil {
		return nil, err
	}
	return stateFromCatalogs(payload.Revision, payload.Items), nil
}

func (c *Client) Register(context.Context, modulemanifest.Manifest) (string, error) {
	return "", fmt.Errorf("%w: module releases are registered on the Registry process", model.ErrUnavailable)
}

func (c *Client) Activate(ctx context.Context, _ *model.RegistryAuditActor, moduleID, version string) error {
	return c.post(ctx, "/activate", map[string]string{"moduleId": moduleID, "version": version})
}

func (c *Client) Deactivate(ctx context.Context, _ *model.RegistryAuditActor, moduleID string) error {
	return c.post(ctx, "/deactivate", map[string]string{"moduleId": moduleID})
}

func stateFromCatalogs(revision uint64, items []model.ModuleCapabilityCatalog) *model.RegistryState {
	state := model.NewRegistryState()
	state.Revision = revision
	if state.Releases == nil {
		state.Releases = map[string]map[string]model.Release{}
	}
	if state.Active == nil {
		state.Active = map[string]string{}
	}
	for _, catalog := range items {
		versions := map[string]model.Release{}
		for _, release := range catalog.Releases {
			versions[release.Version] = model.Release{
				Digest: release.Digest,
				Manifest: modulemanifest.Manifest{
					Metadata: modulemanifest.Metadata{ID: catalog.ID, Name: catalog.Name, Version: release.Version},
					Spec: modulemanifest.Spec{
						Backend:       release.Backend,
						Permissions:   release.Permissions,
						Contributions: release.Contributions,
					},
				},
			}
			if release.Active {
				state.Active[catalog.ID] = release.Version
			}
		}
		state.Releases[catalog.ID] = versions
	}
	return state
}

func (c *Client) get(ctx context.Context, path string, dest any) error {
	return c.do(ctx, http.MethodGet, path, nil, dest)
}

func (c *Client) post(ctx context.Context, path string, body any) error {
	return c.do(ctx, http.MethodPost, path, body, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body, dest any) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.origin+registryPrefix+path, payload)
	if err != nil {
		return err
	}
	token, err := c.issuer.Sign(workloadTokenTTL)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %s", model.ErrUnavailable, err.Error())
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("%w: %s", model.ErrUnavailable, err.Error())
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return mapRegistryStatus(response.StatusCode, raw)
	}
	if dest == nil {
		return nil
	}
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Code != 0 {
		return fmt.Errorf("%w: registry response is invalid", model.ErrUnavailable)
	}
	if len(envelope.Data) == 0 {
		return nil
	}
	return json.Unmarshal(envelope.Data, dest)
}

func mapRegistryStatus(status int, raw []byte) error {
	var failure struct {
		Reason string `json:"reason"`
	}
	_ = json.Unmarshal(raw, &failure)
	switch {
	case strings.Contains(failure.Reason, "release_not_found"):
		return model.ErrReleaseNotFound
	case strings.Contains(failure.Reason, "release_invalid"):
		return model.ErrReleaseInvalid
	case strings.Contains(failure.Reason, "release_immutable"):
		return model.ErrReleaseImmutable
	case strings.Contains(failure.Reason, "route_conflict"):
		return model.ErrRouteConflict
	case strings.Contains(failure.Reason, "navigation_group_conflict"):
		return model.ErrNavigationGroupConflict
	case strings.Contains(failure.Reason, "self_deactivation"):
		return model.ErrPlatformSelfDeactivation
	}
	switch status {
	case http.StatusNotFound:
		return model.ErrReleaseNotFound
	case http.StatusBadRequest:
		return model.ErrReleaseInvalid
	case http.StatusForbidden:
		return model.ErrPlatformSelfDeactivation
	case http.StatusConflict:
		return model.ErrReleaseImmutable
	default:
		return model.ErrUnavailable
	}
}
