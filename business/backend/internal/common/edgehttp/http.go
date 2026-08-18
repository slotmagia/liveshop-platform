// Package edgehttp talks to Identity host resolve and Caddy admin over HTTP.
package edgehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	bizedge "github.com/liveshop-platform/module-platform/internal/biz/capability/edge"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
)

type IdentityHTTP struct {
	origin string
	token  string
	client *http.Client
}

func NewIdentityHTTP(origin, token string) bizedge.IdentityResolver {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return nilResolver{}
	}
	return &IdentityHTTP{origin: origin, token: token, client: &http.Client{Timeout: 5 * time.Second}}
}

type nilResolver struct{}

func (nilResolver) Resolve(context.Context, string, string) (model.Binding, error) {
	return model.Binding{}, model.ErrNotBound
}

func (c *IdentityHTTP) Resolve(ctx context.Context, host, rootDomain string) (model.Binding, error) {
	query := url.Values{"host": {host}}
	if rootDomain != "" {
		query.Set("rootDomain", rootDomain)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.origin+"/internal/v1/domains/resolve?"+query.Encode(), nil)
	if err != nil {
		return model.Binding{}, err
	}
	if c.token != "" {
		request.Header.Set("X-Liveshop-Internal-Grant", c.token)
	}
	response, err := c.client.Do(request)
	if err != nil {
		return model.Binding{}, fmt.Errorf("edge identity resolve: %w", err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return model.Binding{}, err
	}
	if response.StatusCode == http.StatusNotFound {
		return model.Binding{}, model.ErrNotBound
	}
	if response.StatusCode >= 300 {
		return model.Binding{}, fmt.Errorf("edge identity resolve: upstream %d", response.StatusCode)
	}
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	body := payload
	if json.Unmarshal(payload, &envelope) == nil && envelope.Code == 0 && len(envelope.Data) > 0 {
		body = envelope.Data
	}
	var view struct {
		Kind          string `json:"kind"`
		Host          string `json:"host"`
		MerchantID    int64  `json:"merchantId"`
		ShopID        int64  `json:"shopId"`
		ShopStatus    string `json:"shopStatus"`
		Scene         string `json:"scene"`
		DomainStatus  string `json:"domainStatus"`
		OverlayStatus string `json:"overlayStatus"`
	}
	if err := json.Unmarshal(body, &view); err != nil {
		return model.Binding{}, err
	}
	if view.Kind == "" || view.ShopID <= 0 {
		return model.Binding{}, model.ErrNotBound
	}
	return model.Binding{
		Kind: view.Kind, Host: view.Host, MerchantID: view.MerchantID, ShopID: view.ShopID,
		ShopStatus: view.ShopStatus, Scene: view.Scene, DomainStatus: view.DomainStatus, OverlayStatus: view.OverlayStatus,
	}, nil
}

type CaddyReloader struct {
	admin  string
	path   string
	write  func(string, []byte) error
	client *http.Client
}

func NewCaddyReloader(admin, path string, write func(string, []byte) error) bizedge.Reloader {
	if strings.TrimSpace(admin) == "" || strings.TrimSpace(path) == "" || write == nil {
		return nil
	}
	return &CaddyReloader{admin: strings.TrimRight(strings.TrimSpace(admin), "/"), path: path, write: write, client: &http.Client{Timeout: 8 * time.Second}}
}

func (r *CaddyReloader) Load(ctx context.Context, document string) error {
	if err := r.write(r.path, []byte(document)); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.admin+"/load", bytes.NewReader([]byte(document)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/caddyfile")
	response, err := r.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("caddy load %d: %s", response.StatusCode, payload)
	}
	return nil
}
