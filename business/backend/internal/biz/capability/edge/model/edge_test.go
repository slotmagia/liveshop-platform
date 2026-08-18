package model_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
)

func TestParseDomainBaseAndReservedHosts(t *testing.T) {
	base := model.ParseDomainBase(json.RawMessage(`{
		"root_domain":"Wopays.com","shop_domain":"shop.wopays.com","live_domain":"live.wopays.com",
		"rts_domain":"rts.wopays.com","admin_domain":"admin.wopays.com","merchant_domain":"merchant.wopays.com",
		"custom_domain_cname_target":"edge.wopays.com","force_https":true
	}`))
	if base.RootDomain != "wopays.com" || !base.ForceHTTPS || base.CNAMETarget != "edge.wopays.com" {
		t.Fatalf("base=%+v", base)
	}
	target, ok := base.EntryHost("shop.wopays.com")
	if !ok || target != model.TargetShop {
		t.Fatalf("entry=%s ok=%v", target, ok)
	}
	reserved := strings.Join(base.ReservedHosts(), ",")
	if !strings.Contains(reserved, "wopays.com") || !strings.Contains(reserved, "edge.wopays.com") {
		t.Fatalf("reserved=%s", reserved)
	}
}

func TestNormalizeHostRejectsIPAndPath(t *testing.T) {
	if _, err := model.NormalizeHost("https://shop.example.com"); err != model.ErrHostInvalid {
		t.Fatalf("scheme err=%v", err)
	}
	if _, err := model.NormalizeHost("127.0.0.1"); err != model.ErrHostInvalid {
		t.Fatalf("ip err=%v", err)
	}
	host, err := model.NormalizeHost("Shop.Example.COM:443")
	if err != nil || host != "shop.example.com" {
		t.Fatalf("host=%s err=%v", host, err)
	}
}

func TestSlugOfOnlyOneLabel(t *testing.T) {
	if model.SlugOf("acme.wopays.com", "wopays.com") != "acme" {
		t.Fatal("expected slug")
	}
	if model.SlugOf("a.b.wopays.com", "wopays.com") != "" || model.SlugOf("wopays.com", "wopays.com") != "" {
		t.Fatal("nested or root must not slug")
	}
}

func TestCustomEligibility(t *testing.T) {
	binding := model.Binding{Kind: model.KindCustom, DomainStatus: "VERIFIED", OverlayStatus: "active", ShopStatus: "ACTIVE", Scene: "LIVE"}
	if !model.CustomCertAllowed(binding) || model.CustomRouteAllowed(binding) && binding.Scene != "LIVE" {
		t.Fatal("verified live should certify")
	}
	binding.DomainStatus = "PENDING"
	if model.CustomCertAllowed(binding) {
		t.Fatal("pending must not certify")
	}
	binding.DomainStatus = "VERIFIED"
	binding.OverlayStatus = "restricted"
	if model.CustomCertAllowed(binding) {
		t.Fatal("restricted overlay must not certify")
	}
}

func TestRenderCaddyfilePointsAskAtPlatform(t *testing.T) {
	document, err := model.RenderCaddyfile(model.RenderInput{
		Domains: model.DomainBase{
			RootDomain: "wopays.com", ShopDomain: "shop.wopays.com", LiveDomain: "live.wopays.com",
			RTSDomain: "rts.wopays.com", AdminDomain: "admin.wopays.com", MerchantDomain: "merchant.wopays.com",
		},
		ACMEEmail: "ops@example.com",
		AskOrigin: "http://platform:18082",
		Grant:     "secret-token",
		Upstreams: map[string]string{
			model.TargetShop: "shop-host:18080", model.TargetLive: "live-host:18080", model.TargetMerch: "merch-host:18080",
			model.TargetAdmin: "admin-host:18080", model.TargetRTS: "gateway:18081", model.TargetGateway: "gateway:18081",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(document, "ask http://platform:18082/internal/platform/edge/hosts/allowed?grant=secret-token") {
		t.Fatalf("ask missing: %s", document)
	}
	if strings.Contains(document, "aggregator:8081") || !strings.Contains(document, "reverse_proxy gateway:18081") {
		t.Fatalf("old aggregator leaked: %s", document)
	}
	if _, err := model.RenderCaddyfile(model.RenderInput{Domains: model.DomainBase{RootDomain: "wopays.com"}}); err == nil {
		t.Fatal("incomplete domains must fail")
	}
}
