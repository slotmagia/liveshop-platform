package telemetry

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
)

type memoryRepo struct {
	mu    sync.Mutex
	items []model.Event
}

func newMemoryRepo() *memoryRepo { return &memoryRepo{} }

func eventKey(item model.Event) string {
	return strings.Join([]string{fmt.Sprintf("%d", item.MerchantID), fmt.Sprintf("%d", item.ShopID), item.Surface, item.EventID}, "|")
}

func (r *memoryRepo) InsertIgnore(_ context.Context, items []model.Event) ([]bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := map[string]struct{}{}
	for _, item := range r.items {
		seen[eventKey(item)] = struct{}{}
	}
	inserted := make([]bool, len(items))
	for i, item := range items {
		key := eventKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		r.items = append(r.items, item)
		inserted[i] = true
	}
	return inserted, nil
}

func (r *memoryRepo) List(_ context.Context, filter model.Filter) (model.Page, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []model.Event
	for _, item := range r.items {
		if filter.MerchantID > 0 && item.MerchantID != filter.MerchantID {
			continue
		}
		if filter.ShopID > 0 && item.ShopID != filter.ShopID {
			continue
		}
		if filter.Surface != "" && item.Surface != filter.Surface {
			continue
		}
		if filter.EventName != "" && item.EventName != filter.EventName {
			continue
		}
		if filter.EventType != "" && item.EventType != filter.EventType {
			continue
		}
		if filter.Subject != "" && item.Subject != filter.Subject {
			continue
		}
		if filter.AnonymousID != "" && item.AnonymousID != filter.AnonymousID {
			continue
		}
		if filter.StartMs > 0 && item.ClientTs < filter.StartMs {
			continue
		}
		if filter.EndMs > 0 && item.ClientTs > filter.EndMs {
			continue
		}
		matched = append(matched, item)
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].ClientTs != matched[j].ClientTs {
			return matched[i].ClientTs > matched[j].ClientTs
		}
		return matched[i].EventID > matched[j].EventID
	})
	total := len(matched)
	start := filter.Offset()
	if start > total {
		start = total
	}
	end := start + filter.PageSize
	if end > total {
		end = total
	}
	return model.Page{Items: append([]model.Event{}, matched[start:end]...), Total: total}, nil
}
