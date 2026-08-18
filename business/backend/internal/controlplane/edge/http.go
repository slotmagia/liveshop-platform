package edge

import (
	"net/http"

	"github.com/gogf/gf/v2/net/ghttp"
	bizedge "github.com/liveshop-platform/module-platform/internal/biz/capability/edge"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
	"github.com/liveshop-platform/module-platform/internal/common/server"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

const Prefix = "/internal/platform/edge"

type Surface struct {
	token string
	edge  *bizedge.UseCase
}

func New(token string, use *bizedge.UseCase) Surface {
	return Surface{token: token, edge: use}
}

func (s Surface) RegisterHTTP(root *ghttp.RouterGroup) {
	root.Group(Prefix, func(group *ghttp.RouterGroup) {
		group.Middleware(requireGrant(s.token))
		group.GET("/hosts/allowed", s.allowed)
		group.GET("/hosts/route", s.route)
	})
}

var _ server.Surface = Surface{}

func (s Surface) allowed(request *ghttp.Request) {
	if s.edge == nil {
		request.Response.WriteStatus(http.StatusServiceUnavailable)
		return
	}
	host := request.Get("domain").String()
	if host == "" {
		host = request.Get("host").String()
	}
	decision, err := s.edge.Allow(request.Context(), host)
	writeClassic(request, decision, err, false)
}

func (s Surface) route(request *ghttp.Request) {
	if s.edge == nil {
		request.Response.WriteStatus(http.StatusServiceUnavailable)
		return
	}
	host := request.Get("host").String()
	if host == "" {
		host = request.Get("domain").String()
	}
	decision, err := s.edge.Route(request.Context(), host)
	writeClassic(request, decision, err, true)
}

func writeClassic(request *ghttp.Request, decision model.Decision, err error, withUpstream bool) {
	if err != nil {
		if status, ok := web.StatusFor(err); ok {
			request.Response.WriteStatus(status)
			return
		}
		request.Response.WriteStatus(http.StatusInternalServerError)
		return
	}
	if !decision.Allowed {
		status := decision.DenyStatus
		if status == 0 {
			status = http.StatusForbidden
		}
		request.Response.WriteStatus(status)
		return
	}
	if withUpstream {
		request.Response.Header().Set("X-Upstream", decision.Upstream)
		request.Response.Header().Set("X-Liveshop-Edge-Kind", decision.Kind)
		request.Response.Header().Set("X-Liveshop-Edge-Target", decision.Target)
	}
	request.Response.WriteStatus(http.StatusOK)
}

func requireGrant(token string) func(*ghttp.Request) {
	return func(request *ghttp.Request) {
		if token != "" && (request.Header.Get("X-Liveshop-Internal-Grant") == token || request.Get("grant").String() == token) {
			request.Middleware.Next()
			return
		}
		request.Response.WriteStatus(http.StatusUnauthorized)
		request.ExitAll()
	}
}
