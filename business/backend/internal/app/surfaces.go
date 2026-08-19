package app

import (
	"github.com/liveshop-platform/module-platform/internal/application/admin"
	"github.com/liveshop-platform/module-platform/internal/application/internalgrant"
	"github.com/liveshop-platform/module-platform/internal/common/server"
	ctrledge "github.com/liveshop-platform/module-platform/internal/controlplane/edge"
	ctrli18n "github.com/liveshop-platform/module-platform/internal/controlplane/i18n"
	ctrlnotify "github.com/liveshop-platform/module-platform/internal/controlplane/notification"
	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning"
)

// Applications is the assembled set of Platform control-plane surfaces.
type Applications struct {
	Provisioning provisioning.Surface
	Admin        admin.Surface
	Internal     internalgrant.Surface
	Notification ctrlnotify.Surface
	I18n         ctrli18n.Surface
	Edge         ctrledge.Surface
}

func (a Applications) HTTP() []server.Surface {
	return []server.Surface{a.Provisioning, a.Admin, a.Internal, a.Notification, a.I18n, a.Edge}
}

func NewApplications(deps Dependencies, internalToken string) Applications {
	adminSurface := admin.New(admin.Config{
		Release: deps.Release, Settings: deps.Settings, Audit: deps.Audit, LiveProvider: deps.LiveProvider, SMS: deps.SMS, Email: deps.Email, Storage: deps.Storage,
		Notification: deps.Notification, Localization: deps.Localization, Edge: deps.Edge, ModuleSessions: deps.ModuleVerifier,
	})
	return Applications{
		Provisioning: provisioning.New(provisioning.Config{Release: deps.Release, Notification: deps.Notification, Workloads: deps.Workloads}),
		Admin:        adminSurface,
		Internal:     internalgrant.New(internalToken, adminSurface.Application(), adminSurface.Application(), deps.Edge),
		Notification: ctrlnotify.New(ctrlnotify.Config{Notification: deps.Notification, Workloads: deps.Workloads}),
		I18n:         ctrli18n.New(ctrli18n.Config{Localization: deps.Localization, Workloads: deps.Workloads, Grant: internalToken}),
		Edge:         ctrledge.New(internalToken, deps.Edge),
	}
}
