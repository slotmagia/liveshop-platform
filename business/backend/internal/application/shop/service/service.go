package service

import (
	"context"

	telemmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry/model"
)

type TrackEvents interface {
	CreateTrackEvents(context.Context, telemmodel.Scope, []telemmodel.EventInput) (telemmodel.IngestResult, error)
}
