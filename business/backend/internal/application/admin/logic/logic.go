// Package logic implements the admin surface boundary. It reads tenant and
// operator from the verified request context and returns domain errors.
package logic

import (
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/edge"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/email"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/notification"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/sms"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/storage"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/telemetry"
)

// Deps are the use cases the admin surface is allowed to reach.
type Deps struct {
	Release      *biz.Release
	Settings     *biz.Settings
	Audit        *biz.Audit
	LiveProvider *liveprovider.UseCase
	SMS          *sms.UseCase
	Email        *email.UseCase
	Storage      *storage.UseCase
	Notification *notification.UseCase
	Localization *localization.UseCase
	Telemetry    *telemetry.UseCase
	Edge         *edge.UseCase
}

type Logic struct{ deps Deps }

var (
	_ service.Registry        = (*Logic)(nil)
	_ service.Settings        = (*Logic)(nil)
	_ service.Audit           = (*Logic)(nil)
	_ service.LiveProvider    = (*Logic)(nil)
	_ service.SMS             = (*Logic)(nil)
	_ service.Email           = (*Logic)(nil)
	_ service.Storage         = (*Logic)(nil)
	_ service.NotifyEvents    = (*Logic)(nil)
	_ service.NotifyTemplates = (*Logic)(nil)
	_ service.NotifyChannels  = (*Logic)(nil)
	_ service.I18n            = (*Logic)(nil)
	_ service.TrackEvents     = (*Logic)(nil)
)

func New(deps Deps) *Logic { return &Logic{deps: deps} }
