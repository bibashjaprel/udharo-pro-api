package customer

import "net/http"

const CustomersPath = "/api/v1/customers"

func ProtectedRoutes(handler *Handler) map[string]http.Handler {
	return map[string]http.Handler{
		CustomersPath: http.HandlerFunc(handler.Customers),
	}
}
