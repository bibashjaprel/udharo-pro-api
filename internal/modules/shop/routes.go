package shop

import "net/http"

const CurrentShopPath = "/api/v1/shop"

func ProtectedRoutes(handler *Handler) map[string]http.Handler {
	return map[string]http.Handler{
		CurrentShopPath: http.HandlerFunc(handler.CurrentShop),
	}
}
