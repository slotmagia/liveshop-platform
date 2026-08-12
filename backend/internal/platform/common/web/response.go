// Package web 提供 Platform HTTP 统一响应与错误映射。
package web

import (
	"errors"
	"net/http"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/lvtuopen-ai/kernel-go/apperror"
)

type HTTPError struct {
	Status int
	Cause  error
}

func (e *HTTPError) Error() string { return e.Cause.Error() }
func (e *HTTPError) Unwrap() error { return e.Cause }

func Error(status int, err error) error {
	if err == nil {
		err = errors.New(http.StatusText(status))
	}
	return &HTTPError{Status: status, Cause: err}
}

type noContent interface{ NoContent() bool }
type noData interface{ NoData() bool }

func ResponseHandler(request *ghttp.Request) {
	request.Middleware.Next()
	if request.Response.BufferLength() > 0 {
		return
	}
	if err := request.GetError(); err != nil {
		status := http.StatusInternalServerError
		var httpError *HTTPError
		if errors.As(err, &httpError) {
			status = httpError.Status
		}
		WriteFailure(request, status, err)
		return
	}
	data := request.GetHandlerResponse()
	if marker, ok := data.(noContent); ok && marker.NoContent() {
		request.Response.WriteStatus(http.StatusNoContent)
		return
	}
	request.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	if marker, ok := data.(noData); ok && marker.NoData() {
		request.Response.WriteJson(map[string]any{"code": 0})
		return
	}
	request.Response.WriteJson(map[string]any{"code": 0, "data": data})
}

func WriteFailure(request *ghttp.Request, status int, err error) {
	request.Response.ClearBuffer()
	request.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	request.Response.WriteHeader(status)
	response := map[string]any{"code": status * 100, "message": err.Error()}
	if applicationError, ok := apperror.As(err); ok {
		response["reason"] = applicationError.Reason
		if len(applicationError.Args) > 0 {
			response["args"] = applicationError.Args
		}
	}
	request.Response.WriteJson(response)
}
