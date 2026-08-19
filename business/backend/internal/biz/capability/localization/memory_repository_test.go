package localization

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
)

type memoryRepo struct {
	mu       sync.Mutex
	config   model.Config
	apiKey   string
	commands map[string]string
	sources  map[string]model.SourceSnapshot
	texts    map[string]storedText
}

type storedText struct {
	row                  model.WorklistRow
	entityType           string
	locale               string
	sourceVersionAtWrite int64
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{commands: map[string]string{}, sources: map[string]model.SourceSnapshot{}, texts: map[string]storedText{}}
}

func sourceKey(item model.SourceSnapshot) string {
	return strings.Join([]string{item.EntityType, item.EntityID, formatID(item.MerchantID), formatID(item.ShopID)}, "|")
}

func textKey(entityType, entityID, locale string, merchantID, shopID int64) string {
	return strings.Join([]string{entityType, entityID, locale, formatID(merchantID), formatID(shopID)}, "|")
}

func formatID(id int64) string { return fmt.Sprintf("%d", id) }

func (r *memoryRepo) GetConfig(context.Context) (model.Config, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg := r.config
	if cfg.Provider == "" {
		cfg.Provider = model.ProviderNoop
	}
	return cfg, nil
}

func (r *memoryRepo) LoadAPIKey(context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.apiKey, nil
}

func (r *memoryRepo) UpsertConfig(_ context.Context, input model.UpsertConfig, requestHash string) (model.Config, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if stored, ok := r.commands[input.CommandKey]; ok {
		if stored != requestHash {
			return model.Config{}, model.ErrConflict
		}
		return r.config, nil
	}
	if r.config.Version != input.ExpectedVersion {
		return model.Config{}, model.ErrConflict
	}
	r.config.Provider = input.Provider
	if input.APIKeyClear {
		r.apiKey = ""
		r.config.APIKeySet = false
	} else if input.APIKey != "" {
		r.apiKey = input.APIKey
		r.config.APIKeySet = true
	}
	r.config.Version++
	r.commands[input.CommandKey] = requestHash
	return r.config, nil
}

func (r *memoryRepo) ListWorklist(_ context.Context, entityType, locale string) ([]model.WorklistRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []model.WorklistRow{}
	for _, source := range r.sources {
		if source.EntityType != entityType {
			continue
		}
		row := model.WorklistRow{EntityID: source.EntityID, MerchantID: source.MerchantID, ShopID: source.ShopID, Source: source.Source}
		if text, ok := r.texts[textKey(entityType, source.EntityID, locale, source.MerchantID, source.ShopID)]; ok {
			row.Value = text.row.Value
			row.Status = text.row.Status
			row.TextSource = text.row.TextSource
			row.Version = text.row.Version
			row.Stale = source.SourceVersion > text.sourceVersionAtWrite
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *memoryRepo) Publish(_ context.Context, input model.PublishInput, requestHash string) (model.PublishResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if stored, ok := r.commands[input.CommandKey]; ok {
		if stored != requestHash {
			return model.PublishResult{}, model.ErrConflict
		}
		text := r.texts[textKey(input.EntityType, input.EntityID, input.Locale, input.MerchantID, input.ShopID)]
		return model.PublishResult{OK: true, Version: text.row.Version}, nil
	}
	source, ok := r.sources[sourceKey(model.SourceSnapshot{EntityType: input.EntityType, EntityID: input.EntityID, MerchantID: input.MerchantID, ShopID: input.ShopID})]
	if !ok {
		return model.PublishResult{}, model.ErrNotFound
	}
	key := textKey(input.EntityType, input.EntityID, input.Locale, input.MerchantID, input.ShopID)
	current := r.texts[key]
	if current.row.Version != input.ExpectedVersion {
		return model.PublishResult{}, model.ErrConflict
	}
	current.row = model.WorklistRow{EntityID: input.EntityID, MerchantID: input.MerchantID, ShopID: input.ShopID, Source: source.Source, Value: input.Value, Status: model.StatusPublished, TextSource: model.SourceHuman, Version: current.row.Version + 1}
	current.entityType = input.EntityType
	current.locale = input.Locale
	current.sourceVersionAtWrite = source.SourceVersion
	r.texts[key] = current
	r.commands[input.CommandKey] = requestHash
	return model.PublishResult{OK: true, Version: current.row.Version}, nil
}

func (r *memoryRepo) Fill(ctx context.Context, input model.FillInput, provider model.Provider, apiKey string, translator Translator, requestHash string) (model.FillResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if stored, ok := r.commands[input.CommandKey]; ok {
		if stored != requestHash {
			return model.FillResult{}, model.ErrConflict
		}
		return model.FillResult{Provider: provider}, nil
	}
	filled, skipped := 0, 0
	for _, source := range r.sources {
		if source.EntityType != input.EntityType {
			continue
		}
		key := textKey(input.EntityType, source.EntityID, input.Locale, source.MerchantID, source.ShopID)
		current := r.texts[key]
		if current.row.Status == model.StatusPublished && source.SourceVersion <= current.sourceVersionAtWrite {
			skipped++
			continue
		}
		translated, err := translator.Translate(ctx, provider, apiKey, source.Source, input.Locale)
		if err != nil {
			skipped++
			continue
		}
		current.row = model.WorklistRow{EntityID: source.EntityID, MerchantID: source.MerchantID, ShopID: source.ShopID, Source: source.Source, Value: translated, Status: model.StatusMachine, TextSource: model.SourceMachine, Version: current.row.Version + 1}
		current.entityType = input.EntityType
		current.locale = input.Locale
		current.sourceVersionAtWrite = source.SourceVersion
		r.texts[key] = current
		filled++
	}
	r.commands[input.CommandKey] = requestHash
	return model.FillResult{Provider: provider, Filled: filled, Skipped: skipped}, nil
}

func (r *memoryRepo) UpsertSource(_ context.Context, snapshot model.SourceSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[sourceKey(snapshot)] = snapshot
	return nil
}

func (r *memoryRepo) ListPublished(_ context.Context, entityType, locale string, merchantID, shopID int64) ([]model.PublishedText, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []model.PublishedText{}
	for key, text := range r.texts {
		if text.entityType != entityType || text.locale != locale || text.row.Status != model.StatusPublished {
			continue
		}
		if merchantID != 0 || shopID != 0 {
			if text.row.MerchantID != merchantID || text.row.ShopID != shopID {
				continue
			}
		}
		_ = key
		out = append(out, model.PublishedText{EntityID: text.row.EntityID, Value: text.row.Value, Version: text.row.Version})
	}
	return out, nil
}
