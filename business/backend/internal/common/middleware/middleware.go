// Package middleware implements the Platform ghttp transport middleware. It is
// the only place that turns credentials into verified request context.
package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/biz/model"
	"github.com/liveshop-platform/module-platform/internal/common/authctx"
	"github.com/liveshop-platform/module-platform/internal/common/web"
	"github.com/lvtuopen-ai/kernel-go/logctx"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
	"github.com/lvtuopen-ai/kernel-go/principal"
	"github.com/lvtuopen-ai/kernel-go/requestmeta"
	"github.com/lvtuopen-ai/kernel-go/workloadidentity"
)

func RequestMetadata(request *ghttp.Request) {
	requestID := requestmeta.Ensure(request.Request)
	request.Response.Header().Set(requestmeta.HeaderRequestID, requestID)
	request.SetCtx(requestmeta.Context(request.GetCtx(), requestID))
	request.Middleware.Next()
}

func CORS(allowedOrigins map[string]struct{}) ghttp.HandlerFunc {
	return func(request *ghttp.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowedOrigins[origin]; !ok {
				web.WriteFailure(request, http.StatusForbidden, errors.New("origin is not allowed"))
				request.ExitAll()
				return
			}
			request.Response.Header().Set("Access-Control-Allow-Origin", origin)
			request.Response.Header().Set("Access-Control-Allow-Credentials", "true")
			request.Response.Header().Set("Vary", "Origin")
		}
		request.Response.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Liveshop-Surface,X-Locale,X-Ad-Touch-Id,X-Ad-Touch-Type")
		request.Response.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		if request.Method == http.MethodOptions {
			request.Response.WriteStatus(http.StatusNoContent)
			request.ExitAll()
			return
		}
		request.Middleware.Next()
	}
}

func ValidateJSON(request *ghttp.Request) {
	body := bytes.TrimSpace(request.GetBody())
	if len(body) > 0 && strings.Contains(strings.ToLower(request.Header.Get("Content-Type")), "application/json") && !json.Valid(body) {
		web.WriteFailure(request, http.StatusBadRequest, errors.New("invalid JSON request body"))
		request.ExitAll()
		return
	}
	request.Middleware.Next()
}

func Workload(verifier *workloadidentity.Verifier, permission string) ghttp.HandlerFunc {
	return func(request *ghttp.Request) {
		token, ok := workloadidentity.Bearer(request.Header.Get("Authorization"))
		if !ok || verifier == nil {
			logctx.FromContext(request.GetCtx()).Warn("workload authorization denied", "workload", "unknown", "permission", permission, "method", request.Method, "path", request.URL.Path, "decision", "deny")
			reject(request, http.StatusUnauthorized, "workload identity is required")
			return
		}
		claims, err := verifier.Authorize(token, permission)
		if err != nil {
			logctx.FromContext(request.GetCtx()).Warn("workload authorization denied", "workload", "unverified", "permission", permission, "method", request.Method, "path", request.URL.Path, "decision", "deny")
			reject(request, http.StatusForbidden, "workload is not authorized")
			return
		}
		logctx.FromContext(request.GetCtx()).Info("workload authorization allowed", "workload", claims.Subject, "permission", permission, "method", request.Method, "path", request.URL.Path, "decision", "allow")
		request.SetCtx(authctx.WithWorkloadSubject(request.GetCtx(), claims.Subject))
		request.Middleware.Next()
	}
}

func InternalGrantOrWorkload(verifier *workloadidentity.Verifier, grantToken, permission string) ghttp.HandlerFunc {
	workload := Workload(verifier, permission)
	return func(request *ghttp.Request) {
		if grantToken != "" && request.Header.Get("X-Liveshop-Internal-Grant") == grantToken {
			request.Middleware.Next()
			return
		}
		workload(request)
	}
}

// PlatformModule verifies the Module Capability issued by Identity. Platform
// owns no signer or effective-authorization fallback; an invalid, stale-shaped
// or route-incompatible capability is denied before application code runs.
func PlatformModule(verifier *modulesession.Verifier, surface principal.Surface) ghttp.HandlerFunc {
	return func(request *ghttp.Request) {
		token, ok := modulesession.Bearer(request.Header.Get("Authorization"))
		if !ok || verifier == nil {
			reject(request, http.StatusForbidden, "identity module capability is required")
			return
		}
		claims, err := verifier.Verify(token)
		if err != nil || claims.ModuleID != model.PlatformModuleID || claims.Surface != surface.String() || request.Header.Get("X-Liveshop-Surface") != surface.String() || !claims.Realm.AllowsSurface(surface.String()) || !modulesession.AllowsRequest(claims, request.Method, request.URL.Path) {
			reject(request, http.StatusForbidden, "invalid or unauthorized identity module capability")
			return
		}
		request.SetCtx(authctx.WithCapability(request.GetCtx(), claims))
		request.Middleware.Next()
	}
}

func RequirePermission(permission string) ghttp.HandlerFunc {
	return func(request *ghttp.Request) {
		if !modulesession.HasPermissions(authctx.Capability(request.GetCtx()), permission) {
			reject(request, http.StatusForbidden, "required platform permission is not granted")
			return
		}
		request.Middleware.Next()
	}
}

func reject(request *ghttp.Request, status int, message string) {
	web.WriteFailure(request, status, errors.New(message))
	request.ExitAll()
}
