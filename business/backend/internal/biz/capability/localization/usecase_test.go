package localization

import (
	"context"
	"testing"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
)

func TestPublishAndOverlay(t *testing.T) {
	repo := newMemoryRepo()
	use := New(repo, NoopTranslator{})
	ctx := context.Background()
	scope := model.Scope{Realm: "PLATFORM", Subject: "admin-1"}
	if err := use.UpsertSource(ctx, model.SourceSnapshot{EntityType: "catalog.category", EntityID: "9", MerchantID: 1, ShopID: 2, Source: "服装", SourceVersion: 1}); err != nil {
		t.Fatal(err)
	}
	published, err := use.Publish(ctx, scope, model.PublishInput{CommandKey: "pub-1", EntityType: "catalog.category", EntityID: "9", Locale: "en-US", Value: "Apparel", MerchantID: 1, ShopID: 2})
	if err != nil || !published.OK || published.Version != 1 {
		t.Fatalf("publish: %+v %v", published, err)
	}
	items, err := use.ListPublished(ctx, "catalog.category", "en-US", 1, 2)
	if err != nil || len(items) != 1 || items[0].Value != "Apparel" {
		t.Fatalf("overlay: %+v %v", items, err)
	}
	worklist, err := use.ListWorklist(ctx, scope, "catalog.category", "en-US")
	if err != nil || len(worklist) != 1 || worklist[0].Status != model.StatusPublished || worklist[0].Stale {
		t.Fatalf("worklist: %+v %v", worklist, err)
	}
}

func TestFillSkipsFreshPublished(t *testing.T) {
	repo := newMemoryRepo()
	use := New(repo, NoopTranslator{})
	ctx := context.Background()
	scope := model.Scope{Realm: "PLATFORM", Subject: "admin-1"}
	_ = use.UpsertSource(ctx, model.SourceSnapshot{EntityType: "live.gift", EntityID: "3", MerchantID: 1, ShopID: 2, Source: "玫瑰", SourceVersion: 1})
	if _, err := use.Publish(ctx, scope, model.PublishInput{CommandKey: "pub-gift", EntityType: "live.gift", EntityID: "3", Locale: "en-US", Value: "Rose", MerchantID: 1, ShopID: 2}); err != nil {
		t.Fatal(err)
	}
	result, err := use.Fill(ctx, scope, model.FillInput{CommandKey: "fill-1", EntityType: "live.gift", Locale: "en-US"})
	if err != nil || result.Filled != 0 || result.Skipped != 1 {
		t.Fatalf("fill skip: %+v %v", result, err)
	}
}

func TestFillDraftThenStale(t *testing.T) {
	repo := newMemoryRepo()
	use := New(repo, NoopTranslator{})
	ctx := context.Background()
	scope := model.Scope{Realm: "PLATFORM", Subject: "admin-1"}
	_ = use.UpsertSource(ctx, model.SourceSnapshot{EntityType: "trade.payment.channel", EntityID: "wechat", MerchantID: 0, ShopID: 0, Source: "微信支付", SourceVersion: 1})
	filled, err := use.Fill(ctx, scope, model.FillInput{CommandKey: "fill-pay", EntityType: "trade.payment.channel", Locale: "en-US"})
	if err != nil || filled.Filled != 1 {
		t.Fatalf("fill: %+v %v", filled, err)
	}
	_ = use.UpsertSource(ctx, model.SourceSnapshot{EntityType: "trade.payment.channel", EntityID: "wechat", MerchantID: 0, ShopID: 0, Source: "微信", SourceVersion: 2})
	rows, err := use.ListWorklist(ctx, scope, "trade.payment.channel", "en-US")
	if err != nil || len(rows) != 1 || !rows[0].Stale || rows[0].Status != model.StatusMachine {
		t.Fatalf("stale: %+v %v", rows, err)
	}
}

func TestUnknownEntity(t *testing.T) {
	use := New(newMemoryRepo(), NoopTranslator{})
	_, err := use.ListWorklist(context.Background(), model.Scope{Realm: "PLATFORM", Subject: "a"}, "catalog.product", "en-US")
	if err != model.ErrEntityUnknown {
		t.Fatalf("got %v", err)
	}
}
