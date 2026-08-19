// Package router registers the admin surface transports.
package router

import (
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	adminhttp "github.com/liveshop-platform/module-platform/internal/application/admin/controller/http"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	"github.com/liveshop-platform/module-platform/internal/common/middleware"
	"github.com/liveshop-platform/module-platform/internal/common/web"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
	"github.com/lvtuopen-ai/kernel-go/principal"
)

const (
	Prefix         = "/admin/platform"
	RegistryPrefix = Prefix + "/registry"
)

// Application is the set of admin capabilities implemented by the surface
// logic. Each controller below receives only the slice it needs.
type Application interface {
	service.Registry
	service.Settings
	service.Audit
	service.LiveProvider
	service.SMS
	service.Email
	service.Storage
	service.NotifyEvents
	service.NotifyTemplates
	service.NotifyChannels
	service.I18n
	service.TrackEvents
}

type Deps struct {
	Application    Application
	ModuleSessions *modulesession.Verifier
}

// RegisterHTTP mounts every capability behind the Identity Module Capability and
// its own permission, so a controller can never be reached through another
// capability's grant.
func RegisterHTTP(root *ghttp.RouterGroup, deps Deps) {
	bind := func(prefix, permission string, target any) {
		root.Group(prefix, func(group *ghttp.RouterGroup) {
			group.Middleware(web.ResponseHandler)
			group.Middleware(middleware.PlatformModule(deps.ModuleSessions, principal.SurfaceAdmin))
			group.Middleware(middleware.RequirePermission(permission))
			group.Bind(target)
		})
	}
	bind(RegistryPrefix, "platform.registry.manage", adminhttp.NewRegistry(deps.Application))
	bind(Prefix, "platform.settings.read", adminhttp.NewSettingsReader(deps.Application))
	bind(Prefix, "platform.settings.write", adminhttp.NewSettingsWriter(deps.Application))
	bind(Prefix, "platform.audit.read", adminhttp.NewAudit(deps.Application))
	bind(Prefix, "platform.live-provider.read", adminhttp.NewLiveProviderReader(deps.Application))
	bind(Prefix, "platform.live-provider.manage", adminhttp.NewLiveProviderWriter(deps.Application))
	bind(Prefix, "platform.sms.read", adminhttp.NewSMSReader(deps.Application))
	bind(Prefix, "platform.sms.manage", adminhttp.NewSMSWriter(deps.Application))
	bind(Prefix, "platform.email.read", adminhttp.NewEmailReader(deps.Application))
	bind(Prefix, "platform.email.manage", adminhttp.NewEmailWriter(deps.Application))
	bind(Prefix, "platform.storage.read", adminhttp.NewStorageReader(deps.Application))
	bind(Prefix, "platform.storage.manage", adminhttp.NewStorageWriter(deps.Application))
	bind(Prefix, "platform.notify-event.read", adminhttp.NewNotifyEventsReader(deps.Application))
	bind(Prefix, "platform.notify-event.manage", adminhttp.NewNotifyEventsWriter(deps.Application))
	bind(Prefix, "platform.notify-template.read", adminhttp.NewNotifyTemplatesReader(deps.Application))
	bind(Prefix, "platform.notify-template.manage", adminhttp.NewNotifyTemplatesWriter(deps.Application))
	bind(Prefix, "platform.notify-channel.read", adminhttp.NewNotifyChannelsReader(deps.Application))
	bind(Prefix, "platform.notify-channel.manage", adminhttp.NewNotifyChannelsWriter(deps.Application))
	bind(Prefix, "platform.i18n.read", adminhttp.NewI18nReader(deps.Application))
	bind(Prefix, "platform.i18n.manage", adminhttp.NewI18nWriter(deps.Application))
	bind(Prefix, "platform.track-event.read", adminhttp.NewTrackEventsReader(deps.Application))
	root.Group("/uploads", func(group *ghttp.RouterGroup) {
		group.GET("/:folder/:name", servePublicUpload(deps.Application))
	})
}

func servePublicUpload(application service.Storage) func(*ghttp.Request) {
	return func(request *ghttp.Request) {
		key := strings.Trim(request.Get("folder").String(), "/") + "/" + strings.Trim(request.Get("name").String(), "/")
		item, err := application.GetStorageObject(request.Context(), key)
		if err != nil {
			status, ok := web.StatusFor(err)
			if !ok {
				status = 500
			}
			request.Response.WriteStatus(status)
			return
		}
		request.Response.Header().Set("Content-Type", item.ContentType)
		request.Response.Write([]byte(item.Content))
	}
}
