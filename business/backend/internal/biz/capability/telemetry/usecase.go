package telemetry

import (
	"context"
	"time"

	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
	bizmodel "github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/lvtuopen-ai/kernel-go/apperror"
)

type UseCase struct {
	repository Repository
	now        func() time.Time
}

func New(repository Repository) *UseCase {
	return &UseCase{repository: repository, now: time.Now}
}

func (u *UseCase) Ingest(ctx context.Context, scope model.Scope, events []model.EventInput) (model.IngestResult, error) {
	if u.repository == nil {
		return model.IngestResult{}, model.ErrInvalid
	}
	if scope.Now.IsZero() {
		scope.Now = u.now()
	}
	if err := model.ValidateScope(scope); err != nil {
		return model.IngestResult{}, err
	}
	if len(events) == 0 {
		return model.IngestResult{}, apperror.Wrap(model.ErrInvalid, model.ErrInvalid.Reason, "events must contain at least one item")
	}
	if len(events) > model.MaxEvents {
		return model.IngestResult{}, apperror.Wrap(model.ErrInvalid, model.ErrInvalid.Reason, "events cannot exceed 100")
	}
	result := model.IngestResult{}
	pending := make([]model.Event, 0, len(events))
	for index, input := range events {
		item, rowErr := model.NormalizeAndValidate(scope, input)
		if item.EventID == "" {
			item.EventID = bizmodel.NewEventID()
		}
		if rowErr != nil {
			result.Rejected++
			result.Errors = append(result.Errors, model.ItemError{Index: index, EventID: item.EventID, Code: rowErr.Code, Message: rowErr.Message})
			continue
		}
		pending = append(pending, item)
	}
	if len(pending) == 0 {
		return result, nil
	}
	inserted, err := u.repository.InsertIgnore(ctx, pending)
	if err != nil {
		return model.IngestResult{}, err
	}
	for i, item := range pending {
		if i < len(inserted) && !inserted[i] {
			result.Duplicates++
			continue
		}
		result.Accepted++
		result.Stored = append(result.Stored, item)
	}
	return result, nil
}

func (u *UseCase) List(ctx context.Context, filter model.Filter) (model.Page, error) {
	if u.repository == nil {
		return model.Page{}, model.ErrInvalid
	}
	filter = model.NormalizeFilter(filter)
	if filter.Surface != "" && !model.ValidSurface(filter.Surface) {
		return model.Page{}, model.ErrInvalid
	}
	return u.repository.List(ctx, filter)
}
