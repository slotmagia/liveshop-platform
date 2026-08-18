package internalgrant

import (
	"context"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/application/admin/appmodel"
	"github.com/liveshop-platform/module-platform/internal/application/admin/service"
	bizedge "github.com/liveshop-platform/module-platform/internal/biz/capability/edge"
	edgemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
	bizmodel "github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/server"
	"github.com/liveshop-platform/module-platform/internal/common/web"
)

type Surface struct {
	token string
	sms   service.SMS
	live  service.LiveProvider
	edge  *bizedge.UseCase
}

func New(token string, sms service.SMS, live service.LiveProvider, edge *bizedge.UseCase) Surface {
	return Surface{token: token, sms: sms, live: live, edge: edge}
}

func (s Surface) RegisterHTTP(root *ghttp.RouterGroup) {
	root.Group("/internal/v1", func(group *ghttp.RouterGroup) {
		group.Middleware(web.ResponseHandler)
		group.Middleware(requireToken(s.token))
		group.Bind(&controller{sms: s.sms, live: s.live, edge: s.edge})
	})
}

var _ server.Surface = Surface{}

type controller struct {
	sms  service.SMS
	live service.LiveProvider
	edge *bizedge.UseCase
}

type smsGetReq struct {
	g.Meta     `path:"/sms/merchant-grants" method:"get"`
	MerchantID int64 `json:"merchantId" in:"query"`
	ShopID     int64 `json:"shopId" in:"query"`
}
type smsPutReq struct {
	g.Meta          `path:"/sms/merchant-grants" method:"put"`
	CommandKey      string   `json:"commandKey"`
	ExpectedVersion int64    `json:"expectedVersion"`
	MerchantID      int64    `json:"merchantId"`
	ShopID          int64    `json:"shopId"`
	DialCodes       []string `json:"dialCodes"`
}
type liveGetReq struct {
	g.Meta     `path:"/live-providers/assignments" method:"get"`
	MerchantID int64 `json:"merchantId" in:"query"`
}
type livePutReq struct {
	g.Meta          `path:"/live-providers/assignments" method:"put"`
	CommandKey      string                            `json:"commandKey"`
	ExpectedVersion int64                             `json:"expectedVersion"`
	MerchantID      int64                             `json:"merchantId"`
	Providers       []appmodel.LiveProviderAssignment `json:"providers"`
}

type edgeSnapshotReq struct {
	g.Meta `path:"/edge/snapshot" method:"get"`
}

type edgeSnapshotView struct {
	CnameTarget    string            `json:"cnameTarget"`
	RootDomain     string            `json:"rootDomain"`
	ShopDomain     string            `json:"shopDomain"`
	LiveDomain     string            `json:"liveDomain"`
	RtsDomain      string            `json:"rtsDomain"`
	AdminDomain    string            `json:"adminDomain"`
	MerchantDomain string            `json:"merchantDomain"`
	ForceHTTPS     bool              `json:"forceHttps"`
	ReservedHosts  []string          `json:"reservedHosts"`
	Upstreams      map[string]string `json:"upstreams"`
}

type smsRegionOption struct {
	DialCode string `json:"dialCode"`
	Name     string `json:"name"`
	ISO2     string `json:"iso2"`
	Emoji    string `json:"emoji"`
	Enabled  bool   `json:"enabled"`
}
type smsView struct {
	MerchantID   int64             `json:"merchantId"`
	ShopID       int64             `json:"shopId"`
	DialCodes    []string          `json:"dialCodes"`
	Unrestricted bool              `json:"unrestricted"`
	Regions      []smsRegionOption `json:"regions"`
	Version      int64             `json:"version"`
}
type liveView struct {
	MerchantID int64                             `json:"merchantId"`
	Providers  []appmodel.LiveProviderAssignment `json:"providers"`
	Version    int64                             `json:"version"`
}

func (c *controller) GetSMS(ctx context.Context, req *smsGetReq) (*smsView, error) {
	value, err := c.sms.GetSMSMerchantGrant(ctx, req.MerchantID, req.ShopID)
	if err != nil {
		return nil, web.Failure(err)
	}
	view, err := c.projectSMS(ctx, value)
	if err != nil {
		return nil, web.Failure(err)
	}
	return &view, nil
}

func (c *controller) PutSMS(ctx context.Context, req *smsPutReq) (*smsView, error) {
	value, err := c.sms.PutSMSMerchantGrant(ctx, appmodel.PutSMSMerchantGrant{
		CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, MerchantID: req.MerchantID, ShopID: req.ShopID, DialCodes: req.DialCodes,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	view, err := c.projectSMS(ctx, value)
	if err != nil {
		return nil, web.Failure(err)
	}
	return &view, nil
}

func (c *controller) GetLive(ctx context.Context, req *liveGetReq) (*liveView, error) {
	value, err := c.live.GetLiveProviderAssignments(ctx, req.MerchantID)
	if err != nil {
		return nil, web.Failure(err)
	}
	view := projectLive(value)
	return &view, nil
}

func (c *controller) PutLive(ctx context.Context, req *livePutReq) (*liveView, error) {
	value, err := c.live.PutLiveProviderAssignments(ctx, appmodel.PutLiveProviderAssignments{
		CommandKey: req.CommandKey, ExpectedVersion: req.ExpectedVersion, MerchantID: req.MerchantID, Providers: req.Providers,
	})
	if err != nil {
		return nil, web.Failure(err)
	}
	view := projectLive(value)
	return &view, nil
}

func (c *controller) projectSMS(ctx context.Context, value smsmodel.MerchantGrant) (smsView, error) {
	catalog, err := c.sms.ListSMSRegions(ctx, smsmodel.RegionFilter{Lifecycle: smsmodel.LifecycleActive})
	if err != nil {
		return smsView{}, err
	}
	granted := map[string]bool{}
	for _, dial := range value.DialCodes {
		granted[dial] = true
	}
	regions := make([]smsRegionOption, 0, len(catalog))
	for _, item := range catalog {
		if !item.Enabled {
			continue
		}
		regions = append(regions, smsRegionOption{
			DialCode: item.DialCode, Name: item.Name, ISO2: item.ISO2, Emoji: item.Emoji, Enabled: granted[item.DialCode],
		})
	}
	return smsView{MerchantID: value.MerchantID, ShopID: value.ShopID, DialCodes: value.DialCodes, Unrestricted: value.Unrestricted, Regions: regions, Version: value.Version}, nil
}

func (c *controller) GetEdgeSnapshot(ctx context.Context, _ *edgeSnapshotReq) (*edgeSnapshotView, error) {
	if c.edge == nil {
		return nil, web.Failure(bizmodel.ErrUnavailable)
	}
	value, err := c.edge.Snapshot(ctx)
	if err != nil {
		return nil, web.Failure(err)
	}
	view := projectEdge(value)
	return &view, nil
}

func projectEdge(value edgemodel.Snapshot) edgeSnapshotView {
	hosts := append([]string{}, value.ReservedHosts...)
	upstreams := map[string]string{}
	for key, item := range value.Upstreams {
		upstreams[key] = item
	}
	return edgeSnapshotView{
		CnameTarget: value.CNAMETarget, RootDomain: value.RootDomain, ShopDomain: value.ShopDomain,
		LiveDomain: value.LiveDomain, RtsDomain: value.RTSDomain, AdminDomain: value.AdminDomain,
		MerchantDomain: value.MerchantDomain, ForceHTTPS: value.ForceHTTPS, ReservedHosts: hosts, Upstreams: upstreams,
	}
}

func projectLive(value providermodel.AssignmentSet) liveView {
	out := liveView{MerchantID: value.MerchantID, Providers: []appmodel.LiveProviderAssignment{}, Version: value.Version}
	for _, item := range value.Providers {
		out.Providers = append(out.Providers, appmodel.LiveProviderAssignment{ProviderCode: item.ProviderCode, Name: item.Name, Enabled: item.Enabled, Default: item.Default})
	}
	return out
}

func requireToken(token string) func(*ghttp.Request) {
	return func(request *ghttp.Request) {
		if token != "" && request.Header.Get("X-Liveshop-Internal-Grant") == token {
			request.Middleware.Next()
			return
		}
		request.Response.WriteStatus(http.StatusUnauthorized)
		request.ExitAll()
	}
}
