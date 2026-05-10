package shop

import (
	"net/http"

	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

const CurrentShopPath = "/api/v1/shop"

func ProtectedRoutes(handler *Handler) map[string]http.Handler {
	return map[string]http.Handler{
		CurrentShopPath: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				handler.CurrentShop(w, r)
			case http.MethodPatch:
				handler.UpdateShop(w, r)
			default:
				w.Header().Set("Allow", "GET, PATCH")
				response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
			}
		}),
	}
}
