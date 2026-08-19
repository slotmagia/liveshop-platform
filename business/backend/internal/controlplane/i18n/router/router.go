package router

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/localization"
	"github.com/liveshop-platform/module-platform/internal/common/middleware"
	"github.com/liveshop-platform/module-platform/internal/common/web"
	i18nhttp "github.com/liveshop-platform/module-platform/internal/controlplane/i18n/controller/http"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
)

const Prefix = "/internal/platform/i18n"

type Deps struct {
	UseCase   *localization.UseCase
	Workloads *workloadidentity.Verifier
	Grant     string
}

func RegisterHTTP(root *ghttp.RouterGroup, deps Deps) {
	root.Group(Prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(middleware.InternalGrantOrWorkload(deps.Workloads, deps.Grant, "platform.i18n.read"))
		group.Bind(i18nhttp.NewReader(deps.UseCase))
	})
	root.Group(Prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(middleware.InternalGrantOrWorkload(deps.Workloads, deps.Grant, "platform.i18n.ingest"))
		group.Bind(i18nhttp.NewWriter(deps.UseCase))
	})
}
