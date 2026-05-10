package middleware

import (
	"net/http"
	"strings"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/contextx"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

const TenantHeader = "X-Shop-ID"

func Tenant() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			shopID, ok := contextx.GetShopID(r.Context())
			if !ok || strings.TrimSpace(shopID) == "" {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "tenant context is required")
				return
			}

			requestedShopID := strings.TrimSpace(r.Header.Get(TenantHeader))
			if requestedShopID != "" && requestedShopID != shopID {
				response.Error(w, http.StatusForbidden, "forbidden", "cannot access another shop")
				return
			}

			for _, queryShopID := range r.URL.Query()["shop_id"] {
				if strings.TrimSpace(queryShopID) != "" && strings.TrimSpace(queryShopID) != shopID {
					response.Error(w, http.StatusForbidden, "forbidden", "cannot access another shop")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
