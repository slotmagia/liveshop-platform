package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
)

func shopScope(now time.Time) model.Scope {
	return model.Scope{MerchantID: 2001, ShopID: 3001, Surface: model.SurfaceShop, Subject: "customer-9", Now: now}
}

func TestIngestAcceptsAndDedups(t *testing.T) {
	use := New(newMemoryRepo())
	ctx := context.Background()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	first, err := use.Ingest(ctx, shopScope(now), []model.EventInput{{
		EventID: "evt-1", EventName: "page_view", EventType: "page", Page: "/home", OccurredAtMs: now.UnixMilli(),
	}})
	if err != nil || first.Accepted != 1 || first.Duplicates != 0 {
		t.Fatalf("first: %+v %v", first, err)
	}
	again, err := use.Ingest(ctx, shopScope(now), []model.EventInput{{
		EventID: "evt-1", EventName: "page_view", EventType: "page", Page: "/home", OccurredAtMs: now.UnixMilli(),
	}})
	if err != nil || again.Accepted != 0 || again.Duplicates != 1 {
		t.Fatalf("dup: %+v %v", again, err)
	}
}

func TestIngestRejectsTenantMismatchAndForbiddenFields(t *testing.T) {
	use := New(newMemoryRepo())
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	result, err := use.Ingest(context.Background(), shopScope(now), []model.EventInput{
		{EventName: "page_view", EventType: "page", MerchantID: 9, OccurredAtMs: now.UnixMilli()},
		{EventName: "page_view", EventType: "page", AppID: 1, OccurredAtMs: now.UnixMilli()},
		{EventName: "page_view", EventType: "page", UID: 8, OccurredAtMs: now.UnixMilli()},
	})
	if err != nil || result.Rejected != 3 || result.Accepted != 0 {
		t.Fatalf("result: %+v %v", result, err)
	}
	if result.Errors[0].Code != "tenant_mismatch" || result.Errors[1].Code != "forbidden_field" || result.Errors[2].Code != "uid_mismatch" {
		t.Fatalf("codes: %+v", result.Errors)
	}
}

func TestIngestRequiresShopContext(t *testing.T) {
	use := New(newMemoryRepo())
	_, err := use.Ingest(context.Background(), model.Scope{MerchantID: 2001, Surface: model.SurfaceShop}, []model.EventInput{{EventName: "page_view", EventType: "page"}})
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestIngestGeneratesEventID(t *testing.T) {
	use := New(newMemoryRepo())
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	result, err := use.Ingest(context.Background(), shopScope(now), []model.EventInput{{
		EventName: "page_view", EventType: "page", OccurredAtMs: now.UnixMilli(),
	}})
	if err != nil || result.Accepted != 1 || result.Stored[0].EventID == "" || result.Stored[0].Subject != "customer-9" || result.Stored[0].Surface != model.SurfaceShop {
		t.Fatalf("result: %+v %v", result, err)
	}
}

func TestListFiltersAndPages(t *testing.T) {
	use := New(newMemoryRepo())
	ctx := context.Background()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	_, _ = use.Ingest(ctx, shopScope(now), []model.EventInput{
		{EventID: "a", EventName: "page_view", EventType: "page", AnonymousID: "anon-1", OccurredAtMs: now.UnixMilli()},
		{EventID: "b", EventName: "add_to_cart", EventType: "action", AnonymousID: "anon-2", OccurredAtMs: now.UnixMilli() + 1},
	})
	page, err := use.List(ctx, model.Filter{MerchantID: 2001, ShopID: 3001, EventName: "page_view", Page: 1, PageSize: 20})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].EventID != "a" {
		t.Fatalf("page: %+v %v", page, err)
	}
}
