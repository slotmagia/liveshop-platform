package app

import (
	"github.com/liveshop-platform/module-platform/internal/application/admin"
	"github.com/liveshop-platform/module-platform/internal/application/internalgrant"
	"github.com/liveshop-platform/module-platform/internal/common/server"
	ctrledge "github.com/liveshop-platform/module-platform/internal/controlplane/edge"
	ctrlnotify "github.com/liveshop-platform/module-platform/internal/controlplane/notification"
	"github.com/liveshop-platform/module-platform/internal/controlplane/provisioning"
)

// Applications is the assembled set of Platform control-plane surfaces.
type Applications struct {
	Provisioning provisioning.Surface
	Admin        admin.Surface
	Internal     internalgrant.Surface
	Notification ctrlnotify.Surface
	Edge         ctrledge.Surface
}

func (a Applications) HTTP() []server.Surface {
	return []server.Surface{a.Provisioning, a.Admin, a.Internal, a.Notification, a.Edge}
}

func NewApplications(deps Dependencies, internalToken string) Applications {
	adminSurface := admin.New(admin.Config{
		Release: deps.Release, Settings: deps.Settings, Audit: deps.Audit, LiveProvider: deps.LiveProvider, SMS: deps.SMS, Email: deps.Email, Storage: deps.Storage,
		Notification: deps.Notification, Edge: deps.Edge, ModuleSessions: deps.ModuleVerifier,
	})
	return Applications{
		Provisioning: provisioning.New(provisioning.Config{Release: deps.Release, Notification: deps.Notification, Workloads: deps.Workloads}),
		Admin:        adminSurface,
		Internal:     internalgrant.New(internalToken, adminSurface.Application(), adminSurface.Application(), deps.Edge),
		Notification: ctrlnotify.New(ctrlnotify.Config{Notification: deps.Notification, Workloads: deps.Workloads}),
		Edge:         ctrledge.New(internalToken, deps.Edge),
	}
}
