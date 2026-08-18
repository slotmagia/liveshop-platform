package edge

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
	bizmodel "github.com/liveshop-platform/module-platform/internal/biz/model"
)

type stubSettings struct {
	value json.RawMessage
}

func (s stubSettings) List(context.Context, bizmodel.SettingScope) ([]bizmodel.SettingDocument, error) {
	return nil, nil
}
func (s stubSettings) Get(context.Context, bizmodel.SettingScope, string) (bizmodel.SettingDocument, error) {
	return bizmodel.SettingDocument{Namespace: model.NamespaceBase, Value: s.value, Version: 1}, nil
}
func (s stubSettings) Put(context.Context, bizmodel.SettingScope, string, int64, []byte) (bizmodel.SettingDocument, error) {
	return bizmodel.SettingDocument{}, errors.New("unused")
}

type stubIdentity struct {
	binding model.Binding
	err     error
}

func (s stubIdentity) Resolve(context.Context, string, string) (model.Binding, error) {
	return s.binding, s.err
}

func sampleBase() json.RawMessage {
	return json.RawMessage(`{
		"root_domain":"wopays.com","shop_domain":"shop.wopays.com","live_domain":"live.wopays.com",
		"rts_domain":"rts.wopays.com","admin_domain":"admin.wopays.com","merchant_domain":"merchant.wopays.com",
		"custom_domain_cname_target":"edge.wopays.com","force_https":true
	}`)
}

func sampleUse(identity IdentityResolver) *UseCase {
	return New(biz.NewSettings(stubSettings{value: sampleBase()}), identity, nil, Config{
		Upstreams: map[string]string{
			model.TargetShop: "shop-host:18080", model.TargetLive: "live-host:18080", model.TargetMerch: "merch-host:18080",
			model.TargetAdmin: "admin-host:18080", model.TargetRTS: "gateway:18081", model.TargetGateway: "gateway:18081",
		},
	})
}

func TestAllowPlatformEntryHost(t *testing.T) {
	use := sampleUse(nil)
	got, err := use.Allow(context.Background(), "Shop.Wopays.COM")
	if err != nil || !got.Allowed || got.Target != model.TargetShop || got.Upstream != "shop-host:18080" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func TestAllowRejectsUnboundAndReserved(t *testing.T) {
	use := sampleUse(stubIdentity{err: model.ErrNotBound})
	got, err := use.Allow(context.Background(), "unknown.example.com")
	if err != nil || got.Allowed || got.DenyStatus != model.DenyNotFound {
		t.Fatalf("unbound got=%+v err=%v", got, err)
	}
	got, err = use.Allow(context.Background(), "edge.wopays.com")
	if err != nil || got.Allowed || got.DenyStatus != model.DenyNotFound {
		t.Fatalf("cname target got=%+v err=%v", got, err)
	}
}

func TestAllowCustomRequiresVerified(t *testing.T) {
	use := sampleUse(stubIdentity{binding: model.Binding{
		Kind: model.KindCustom, ShopID: 3, ShopStatus: "ACTIVE", Scene: "SHOP", DomainStatus: "PENDING", OverlayStatus: "active",
	}})
	got, err := use.Allow(context.Background(), "shop.brand.com")
	if err != nil || got.Allowed || got.DenyStatus != model.DenyForbidden {
		t.Fatalf("pending got=%+v err=%v", got, err)
	}
	use = sampleUse(stubIdentity{binding: model.Binding{
		Kind: model.KindCustom, ShopID: 3, ShopStatus: "ACTIVE", Scene: "LIVE", DomainStatus: "VERIFIED", OverlayStatus: "active",
	}})
	got, err = use.Allow(context.Background(), "live.brand.com")
	if err != nil || !got.Allowed || got.Target != model.TargetLive {
		t.Fatalf("verified got=%+v err=%v", got, err)
	}
}

func TestRouteSlugGoesToShop(t *testing.T) {
	use := sampleUse(stubIdentity{binding: model.Binding{Kind: model.KindSlug, ShopID: 3, ShopStatus: "ACTIVE"}})
	got, err := use.Route(context.Background(), "acme.wopays.com")
	if err != nil || !got.Allowed || got.Kind != model.KindSlug || got.Target != model.TargetShop {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	unbound := sampleUse(stubIdentity{err: model.ErrNotBound})
	got, err = unbound.Route(context.Background(), "missing.wopays.com")
	if err != nil || got.Allowed || got.DenyStatus != model.DenyForbidden {
		t.Fatalf("unbound route got=%+v err=%v", got, err)
	}
}

func TestSnapshotPublishesCNAME(t *testing.T) {
	snap, err := sampleUse(nil).Snapshot(context.Background())
	if err != nil || snap.CNAMETarget != "edge.wopays.com" || snap.ShopDomain != "shop.wopays.com" {
		t.Fatalf("snap=%+v err=%v", snap, err)
	}
}

func TestApplyNoopsWhenDisabled(t *testing.T) {
	if err := sampleUse(nil).Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
}
