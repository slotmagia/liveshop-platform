package localization

import (
	"context"
	"strings"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
)

// NoopTranslator copies source text. DeepL/Google keys are accepted but the
// first slice does not call those networks from unit tests; production Fill
// still uses this for provider=noop and as a fallback when the HTTP translator
// is not wired.
type NoopTranslator struct{}

func (NoopTranslator) Translate(_ context.Context, provider model.Provider, _, text, _ string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", model.ErrInvalid
	}
	if provider != model.ProviderNoop && provider != model.ProviderDeepL && provider != model.ProviderGoogle {
		return "", model.ErrInvalid
	}
	return text, nil
}
