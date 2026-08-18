package edgehttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
)

func TestIdentityHTTPEmptyOriginIsUnbound(t *testing.T) {
	resolver := NewIdentityHTTP("  ", "")
	_, err := resolver.Resolve(context.Background(), "shop.example", "example")
	if !errors.Is(err, model.ErrNotBound) {
		t.Fatalf("err=%v", err)
	}
}

func TestIdentityHTTPUnwrapsEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/internal/v1/domains/resolve" || request.URL.Query().Get("host") != "acme.example" {
			t.Errorf("path=%s query=%s", request.URL.Path, request.URL.RawQuery)
		}
		if request.Header.Get("X-Liveshop-Internal-Grant") != "grant-token" {
			t.Errorf("grant=%s", request.Header.Get("X-Liveshop-Internal-Grant"))
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"kind": "slug", "host": "acme.example", "merchantId": 7, "shopId": 9, "shopStatus": "ACTIVE",
			},
		})
	}))
	defer server.Close()
	binding, err := NewIdentityHTTP(server.URL, "grant-token").Resolve(context.Background(), "acme.example", "example")
	if err != nil || binding.Kind != model.KindSlug || binding.ShopID != 9 {
		t.Fatalf("binding=%+v err=%v", binding, err)
	}
}

func TestIdentityHTTPNotFoundIsUnbound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	_, err := NewIdentityHTTP(server.URL, "").Resolve(context.Background(), "missing.example", "example")
	if !errors.Is(err, model.ErrNotBound) {
		t.Fatalf("err=%v", err)
	}
}

func TestCaddyReloaderWritesThenPosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Caddyfile")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/load" || request.Header.Get("Content-Type") != "text/caddyfile" {
			t.Errorf("path=%s type=%s", request.URL.Path, request.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != "site {\n}" {
			t.Errorf("body=%q", body)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	reloader := NewCaddyReloader(server.URL, path, func(name string, data []byte) error {
		return os.WriteFile(name, data, 0o600)
	})
	if reloader == nil {
		t.Fatal("expected reloader")
	}
	if err := reloader.Load(context.Background(), "site {\n}"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "site {\n}" {
		t.Fatalf("file=%q err=%v", got, err)
	}
}
