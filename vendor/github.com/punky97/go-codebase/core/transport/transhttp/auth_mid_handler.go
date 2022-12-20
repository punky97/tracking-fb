package transhttp

import (
	"github.com/punky97/go-codebase/core/logger"
	"net/http"
)

type AuthMiddleware struct {
	route Route
}

func NewAuthMiddleware(route Route) *AuthMiddleware {
	return &AuthMiddleware{
		route: route,
	}
}

func (h *AuthMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	logger.BkLog.Info("authinfo ", h.route.AuthInfo.Enable)
	next(rw, r)
}
