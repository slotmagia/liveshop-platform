package model

import (
	"encoding/json"
	"errors"
	"net"
	"strings"
)

var (
	ErrHostInvalid = errors.New("edge host is invalid")
	ErrNotBound    = errors.New("edge host is not bound")
	ErrForbidden   = errors.New("edge host is not eligible")
	ErrApply       = errors.New("edge caddy apply failed")
)

const (
	TargetShop     = "shop"
	TargetLive     = "live"
	TargetMerch    = "merch"
	TargetAdmin    = "admin"
	TargetRTS      = "rts"
	TargetGateway  = "gateway"
	KindPlatform   = "platform"
	KindCustom     = "custom"
	KindSlug       = "slug"
	NamespaceBase  = "domain-base"
	DenyBadRequest = 400
	DenyNotFound   = 404
	DenyForbidden  = 403
	maxHostLen     = 253
)

type DomainBase struct {
	RootDomain     string
	ShopDomain     string
	LiveDomain     string
	RTSDomain      string
	AdminDomain    string
	MerchantDomain string
	CNAMETarget    string
	ForceHTTPS     bool
}

type Binding struct {
	Kind          string
	Host          string
	MerchantID    int64
	ShopID        int64
	ShopStatus    string
	Scene         string
	DomainStatus  string
	OverlayStatus string
}

type Decision struct {
	Allowed    bool
	DenyStatus int
	Kind       string
	Target     string
	Upstream   string
}

func ParseDomainBase(value json.RawMessage) DomainBase {
	var raw map[string]any
	if len(value) > 0 {
		_ = json.Unmarshal(value, &raw)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	return DomainBase{
		RootDomain:     asHost(raw["root_domain"]),
		ShopDomain:     asHost(raw["shop_domain"]),
		LiveDomain:     asHost(raw["live_domain"]),
		RTSDomain:      asHost(raw["rts_domain"]),
		AdminDomain:    asHost(raw["admin_domain"]),
		MerchantDomain: asHost(raw["merchant_domain"]),
		CNAMETarget:    asHost(raw["custom_domain_cname_target"]),
		ForceHTTPS:     asBool(raw["force_https"]),
	}
}

func (d DomainBase) EntryHost(host string) (string, bool) {
	switch host {
	case d.ShopDomain:
		return TargetShop, true
	case d.LiveDomain:
		return TargetLive, true
	case d.MerchantDomain:
		return TargetMerch, true
	case d.AdminDomain:
		return TargetAdmin, true
	case d.RTSDomain:
		return TargetRTS, true
	default:
		return "", false
	}
}

func (d DomainBase) ReservedHosts() []string {
	seen := map[string]bool{}
	out := make([]string, 0, 7)
	for _, host := range []string{d.RootDomain, d.ShopDomain, d.LiveDomain, d.RTSDomain, d.AdminDomain, d.MerchantDomain, d.CNAMETarget} {
		if host == "" || seen[host] {
			continue
		}
		seen[host] = true
		out = append(out, host)
	}
	return out
}

func (d DomainBase) Snapshot(upstreams map[string]string) Snapshot {
	return Snapshot{
		CNAMETarget:    d.CNAMETarget,
		RootDomain:     d.RootDomain,
		ShopDomain:     d.ShopDomain,
		LiveDomain:     d.LiveDomain,
		RTSDomain:      d.RTSDomain,
		AdminDomain:    d.AdminDomain,
		MerchantDomain: d.MerchantDomain,
		ForceHTTPS:     d.ForceHTTPS,
		ReservedHosts:  d.ReservedHosts(),
		Upstreams:      upstreams,
	}
}

type Snapshot struct {
	CNAMETarget    string
	RootDomain     string
	ShopDomain     string
	LiveDomain     string
	RTSDomain      string
	AdminDomain    string
	MerchantDomain string
	ForceHTTPS     bool
	ReservedHosts  []string
	Upstreams      map[string]string
}

func NormalizeHost(value string) (string, error) {
	host := strings.ToLower(strings.TrimSpace(value))
	if i := strings.Index(host, ":"); i > 0 {
		if _, _, err := net.SplitHostPort(host); err == nil {
			host = host[:i]
		}
	}
	host = strings.TrimSuffix(host, ".")
	if host == "" || strings.Contains(host, "://") || strings.ContainsAny(host, "/\\ ") || strings.Contains(host, "..") {
		return "", ErrHostInvalid
	}
	if len(host) < 4 || len(host) > maxHostLen || !strings.Contains(host, ".") {
		return "", ErrHostInvalid
	}
	if ip := net.ParseIP(host); ip != nil {
		return "", ErrHostInvalid
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", ErrHostInvalid
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return "", ErrHostInvalid
			}
		}
	}
	return host, nil
}

func SlugOf(host, root string) string {
	root = strings.ToLower(strings.TrimSpace(root))
	if root == "" || host == root || !strings.HasSuffix(host, "."+root) {
		return ""
	}
	slug := strings.TrimSuffix(host, "."+root)
	if slug == "" || strings.Contains(slug, ".") {
		return ""
	}
	return slug
}

func OverlayAllows(status string) bool {
	return status == "" || status == "active"
}

func ShopOpen(status string) bool {
	return status != "" && !strings.EqualFold(status, "CLOSED")
}

func CustomCertAllowed(b Binding) bool {
	return b.Kind == KindCustom && b.DomainStatus == "VERIFIED" && OverlayAllows(b.OverlayStatus) && ShopOpen(b.ShopStatus)
}

func CustomRouteAllowed(b Binding) bool {
	return CustomCertAllowed(b) && (b.Scene == "SHOP" || b.Scene == "LIVE")
}

func SlugAllowed(b Binding) bool {
	return b.Kind == KindSlug && ShopOpen(b.ShopStatus)
}

func asHost(raw any) string {
	text, _ := raw.(string)
	host, err := NormalizeHost(text)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(text))
	}
	return host
}

func asBool(raw any) bool {
	switch current := raw.(type) {
	case bool:
		return current
	case string:
		switch strings.ToLower(strings.TrimSpace(current)) {
		case "true", "1", "yes":
			return true
		}
	}
	return false
}
