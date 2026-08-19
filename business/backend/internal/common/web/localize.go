package web

import (
	"errors"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	platformi18n "github.com/liveshop-platform/module-platform/i18n"
	"github.com/lvtuopen-ai/kernel-go/apperror"
)

func localizeFailure(request *ghttp.Request, err error) (message, reason string, args map[string]any) {
	cause := err
	var httpError *HTTPError
	if errors.As(err, &httpError) && httpError.Cause != nil {
		cause = httpError.Cause
	}
	message = cause.Error()
	applicationError, ok := apperror.As(cause)
	if !ok {
		return message, "", nil
	}
	reason = applicationError.Reason
	if len(applicationError.Args) > 0 {
		args = make(map[string]any, len(applicationError.Args))
		for key, value := range applicationError.Args {
			args[key] = value
		}
	}
	if text := platformi18n.Lookup(requestLocale(request), reason); text != "" {
		message = text
	}
	return message, reason, args
}

func requestLocale(request *ghttp.Request) string {
	raw := strings.TrimSpace(request.Header.Get("X-Locale"))
	if raw == "" {
		raw = request.Header.Get("Accept-Language")
	}
	raw = strings.TrimSpace(strings.Split(raw, ",")[0])
	lower := strings.ToLower(strings.ReplaceAll(raw, "_", "-"))
	if strings.HasPrefix(lower, "en") {
		return "en-US"
	}
	return "zh-CN"
}
