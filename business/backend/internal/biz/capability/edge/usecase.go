package edge

import (
	"context"
	"errors"
	"strings"

	"github.com/liveshop-platform/module-platform/internal/biz"
	"github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
	bizmodel "github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/lvtuopen-ai/kernel-go/principal"
)

type IdentityResolver interface {
	Resolve(ctx context.Context, host, rootDomain string) (model.Binding, error)
}

type Reloader interface {
	Load(ctx context.Context, document string) error
}

type Config struct {
	Enabled       bool
	GrantToken    string
	AskOrigin     string
	ACMEEmail     string
	CaddyfilePath string
	Upstreams     map[string]string
}

type UseCase struct {
	settings *biz.Settings
	identity IdentityResolver
	reloader Reloader
	config   Config
}

func New(settings *biz.Settings, identity IdentityResolver, reloader Reloader, config Config) *UseCase {
	upstreams := map[string]string{}
	for key, value := range config.Upstreams {
		upstreams[key] = strings.TrimSpace(value)
	}
	config.Upstreams = upstreams
	return &UseCase{settings: settings, identity: identity, reloader: reloader, config: config}
}

func platformScope() bizmodel.SettingScope {
	return bizmodel.SettingScope{Realm: principal.RealmPlatform.String(), MerchantID: 0, Subject: "platform-edge"}
}

func (u *UseCase) domainBase(ctx context.Context) (model.DomainBase, error) {
	if u == nil || u.settings == nil {
		return model.DomainBase{}, bizmodel.ErrUnavailable
	}
	document, err := u.settings.Get(ctx, platformScope(), model.NamespaceBase)
	if err != nil {
		return model.DomainBase{}, err
	}
	return model.ParseDomainBase(document.Value), nil
}

func (u *UseCase) Snapshot(ctx context.Context) (model.Snapshot, error) {
	base, err := u.domainBase(ctx)
	if err != nil {
		return model.Snapshot{}, err
	}
	return base.Snapshot(u.config.Upstreams), nil
}

func (u *UseCase) Allow(ctx context.Context, rawHost string) (model.Decision, error) {
	decision, err := u.decide(ctx, rawHost, true)
	if err != nil {
		return model.Decision{}, err
	}
	return decision, nil
}

func (u *UseCase) Route(ctx context.Context, rawHost string) (model.Decision, error) {
	return u.decide(ctx, rawHost, false)
}

func (u *UseCase) decide(ctx context.Context, rawHost string, forCert bool) (model.Decision, error) {
	host, err := model.NormalizeHost(rawHost)
	if err != nil {
		return model.Decision{DenyStatus: model.DenyBadRequest}, nil
	}
	base, err := u.domainBase(ctx)
	if err != nil {
		return model.Decision{}, err
	}
	if target, ok := base.EntryHost(host); ok {
		return u.hit(model.KindPlatform, target), nil
	}
	if host == base.RootDomain || (base.CNAMETarget != "" && host == base.CNAMETarget) {
		return model.Decision{DenyStatus: denyStatus(forCert)}, nil
	}
	binding, err := u.resolveIdentity(ctx, host, base.RootDomain)
	if err != nil {
		return model.Decision{}, err
	}
	switch {
	case forCert && model.CustomCertAllowed(binding):
		return u.hit(model.KindCustom, customTarget(binding.Scene)), nil
	case !forCert && model.CustomRouteAllowed(binding):
		return u.hit(model.KindCustom, customTarget(binding.Scene)), nil
	case model.SlugAllowed(binding):
		return u.hit(model.KindSlug, model.TargetShop), nil
	case forCert && binding.Kind == model.KindCustom:
		return model.Decision{DenyStatus: model.DenyForbidden}, nil
	default:
		return model.Decision{DenyStatus: denyStatus(forCert)}, nil
	}
}

func (u *UseCase) resolveIdentity(ctx context.Context, host, root string) (model.Binding, error) {
	if u.identity == nil {
		return model.Binding{}, nil
	}
	binding, err := u.identity.Resolve(ctx, host, root)
	if err != nil {
		if errors.Is(err, model.ErrNotBound) {
			return model.Binding{}, nil
		}
		return model.Binding{}, err
	}
	return binding, nil
}

func (u *UseCase) hit(kind, target string) model.Decision {
	return model.Decision{
		Allowed:  true,
		Kind:     kind,
		Target:   target,
		Upstream: u.config.Upstreams[target],
	}
}

func (u *UseCase) Apply(ctx context.Context) error {
	if u == nil || !u.config.Enabled {
		return nil
	}
	if u.reloader == nil {
		return model.ErrApply
	}
	base, err := u.domainBase(ctx)
	if err != nil {
		return err
	}
	document, err := model.RenderCaddyfile(model.RenderInput{
		Domains:   base,
		ACMEEmail: u.config.ACMEEmail,
		AskOrigin: u.config.AskOrigin,
		Grant:     u.config.GrantToken,
		Upstreams: u.config.Upstreams,
	})
	if err != nil {
		return err
	}
	if err := u.reloader.Load(ctx, document); err != nil {
		return model.ErrApply
	}
	return nil
}

func customTarget(scene string) string {
	if scene == "LIVE" {
		return model.TargetLive
	}
	return model.TargetShop
}

func denyStatus(forCert bool) int {
	if forCert {
		return model.DenyNotFound
	}
	return model.DenyForbidden
}
