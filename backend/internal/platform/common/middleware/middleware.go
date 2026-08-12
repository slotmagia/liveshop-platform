// Package middleware 实现 Platform ghttp 传输中间件。
package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/liveshop-platform/module-platform/internal/platform/common/requestctx"
	"github.com/liveshop-platform/module-platform/internal/platform/common/web"
	"github.com/liveshop-platform/module-platform/internal/platform/registry/iam"
	"github.com/lvtuopen-ai/kernel-go/accessidentity"
	"github.com/lvtuopen-ai/kernel-go/logctx"
	"github.com/lvtuopen-ai/kernel-go/modulesession"
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
		request.Response.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Liveshop-Surface")
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

func UserIdentity(verifier *accessidentity.Verifier, authorizations *iam.Store) ghttp.HandlerFunc {
	return func(request *ghttp.Request) {
		token, ok := accessidentity.Bearer(request.Header.Get("Authorization"))
		if !ok || verifier == nil {
			reject(request, http.StatusUnauthorized, "unauthorized identity")
			return
		}
		identity, err := verifier.Verify(token)
		if err != nil {
			reject(request, http.StatusUnauthorized, "unauthorized identity")
			return
		}
		if authorizations == nil {
			reject(request, http.StatusServiceUnavailable, "authorization service is unavailable")
			return
		}
		authorization, err := authorizations.Effective(request.GetCtx(), iam.Tenant{AppID: identity.AppID, MerchantID: identity.MerchantID}, identity.Subject)
		if err != nil {
			reject(request, http.StatusServiceUnavailable, "authorization service is unavailable")
			return
		}
		request.SetCtx(requestctx.With(request.GetCtx(), identity, authorization))
		request.Middleware.Next()
	}
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
		request.Middleware.Next()
	}
}

func PlatformModule(verifier *modulesession.Verifier) ghttp.HandlerFunc {
	return func(request *ghttp.Request) {
		token, ok := modulesession.Bearer(request.Header.Get("Authorization"))
		if !ok || verifier == nil {
			reject(request, http.StatusForbidden, "platform module session is required")
			return
		}
		claims, err := verifier.Verify(token)
		if err != nil || claims.ModuleID != "platform" || claims.Realm != accessidentity.RealmPlatform || claims.Surface != "admin" || request.Header.Get("X-Liveshop-Surface") != "admin" || !modulesession.AllowsRequest(claims, request.Method, request.URL.Path) {
			reject(request, http.StatusForbidden, "invalid or unauthorized platform module session")
			return
		}
		identity := accessidentity.Claims{Subject: claims.Subject, Realm: claims.Realm, AppID: claims.AppID, MerchantID: claims.MerchantID}
		authorization := iam.Authorization{Revision: claims.AuthorizationRevision, Permissions: claims.Permissions, DataScopes: claims.DataScopes}
		request.SetCtx(requestctx.With(request.GetCtx(), identity, authorization))
		request.Middleware.Next()
	}
}

func RequirePermission(permission string) ghttp.HandlerFunc {
	return func(request *ghttp.Request) {
		if !requestctx.Authorization(request.GetCtx()).Has(permission) {
			reject(request, http.StatusForbidden, "required platform permission is not granted")
			return
		}
		request.Middleware.Next()
	}
}

func RequirePlatformCapabilityReader(request *ghttp.Request) {
	if requestctx.Identity(request.GetCtx()).Realm != accessidentity.RealmPlatform || !requestctx.Authorization(request.GetCtx()).Has("platform.registry.manage") {
		reject(request, http.StatusForbidden, "platform module capability access is required")
		return
	}
	request.Middleware.Next()
}

func reject(request *ghttp.Request, status int, message string) {
	web.WriteFailure(request, status, errors.New(message))
	request.ExitAll()
}
