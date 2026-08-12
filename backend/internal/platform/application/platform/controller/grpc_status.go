package controller

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
)

func grpcError(err error) error {
	if err == nil {
		return nil
	}
	code := codes.Internal
	var httpError *web.HTTPError
	if errors.As(err, &httpError) {
		switch httpError.Status {
		case http.StatusBadRequest:
			code = codes.InvalidArgument
		case http.StatusUnauthorized:
			code = codes.Unauthenticated
		case http.StatusForbidden:
			code = codes.PermissionDenied
		case http.StatusNotFound:
			code = codes.NotFound
		case http.StatusConflict:
			code = codes.Aborted
		case http.StatusPreconditionFailed:
			code = codes.FailedPrecondition
		case http.StatusServiceUnavailable:
			code = codes.Unavailable
		}
	}
	return status.Error(code, err.Error())
}
