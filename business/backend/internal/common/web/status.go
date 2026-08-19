package web

import (
	"errors"
	"net/http"

	edgemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/edge/model"
	providermodel "github.com/liveshop-platform/module-platform/internal/biz/capability/liveprovider/model"
	locmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/localization/model"
	notifymodel "github.com/liveshop-platform/module-platform/internal/biz/capability/notification/model"
	smsmodel "github.com/liveshop-platform/module-platform/internal/biz/capability/sms/model"
	storagemodel "github.com/liveshop-platform/module-platform/internal/biz/capability/storage/model"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
)

// domainStatus is the single HTTP projection of the domain outcomes. gRPC has
// its own projection; neither transport derives its codes from the other.
var domainStatus = []struct {
	err    error
	status int
}{
	{model.ErrReleaseNotFound, http.StatusNotFound},
	{model.ErrReleaseInvalid, http.StatusBadRequest},
	{model.ErrReleaseImmutable, http.StatusConflict},
	{model.ErrRouteConflict, http.StatusConflict},
	{model.ErrNavigationGroupConflict, http.StatusConflict},
	{model.ErrPlatformSelfDeactivation, http.StatusForbidden},

	{model.ErrSettingsInvalid, http.StatusBadRequest},
	{model.ErrSettingsConflict, http.StatusConflict},
	{providermodel.ErrInvalid, http.StatusBadRequest},
	{providermodel.ErrNotFound, http.StatusNotFound},
	{providermodel.ErrConflict, http.StatusConflict},
	{providermodel.ErrRetired, http.StatusConflict},
	{storagemodel.ErrInvalid, http.StatusBadRequest},
	{storagemodel.ErrNotFound, http.StatusNotFound},
	{storagemodel.ErrConflict, http.StatusConflict},
	{storagemodel.ErrRetired, http.StatusConflict},
	{storagemodel.ErrDisabled, http.StatusConflict},
	{smsmodel.ErrInvalid, http.StatusBadRequest},
	{smsmodel.ErrNotFound, http.StatusNotFound},
	{smsmodel.ErrConflict, http.StatusConflict},
	{smsmodel.ErrRetired, http.StatusConflict},
	{smsmodel.ErrInUse, http.StatusConflict},
	{smsmodel.ErrNoChannel, http.StatusConflict},
	{notifymodel.ErrInvalid, http.StatusBadRequest},
	{notifymodel.ErrNotFound, http.StatusNotFound},
	{notifymodel.ErrConflict, http.StatusConflict},
	{notifymodel.ErrForbidden, http.StatusForbidden},
	{edgemodel.ErrHostInvalid, http.StatusBadRequest},
	{edgemodel.ErrNotBound, http.StatusNotFound},
	{edgemodel.ErrForbidden, http.StatusForbidden},
	{edgemodel.ErrApply, http.StatusServiceUnavailable},
	{locmodel.ErrInvalid, http.StatusBadRequest},
	{locmodel.ErrEntityUnknown, http.StatusBadRequest},
	{locmodel.ErrLocaleUnknown, http.StatusBadRequest},
	{locmodel.ErrProviderKey, http.StatusConflict},
	{locmodel.ErrNotFound, http.StatusNotFound},
	{locmodel.ErrConflict, http.StatusConflict},
	{locmodel.ErrUnavailable, http.StatusServiceUnavailable},

	{model.ErrUnavailable, http.StatusServiceUnavailable},
}

// StatusFor reports the HTTP status of a known domain error.
func StatusFor(err error) (int, bool) {
	for _, entry := range domainStatus {
		if errors.Is(err, entry.err) {
			return entry.status, true
		}
	}
	return 0, false
}
