package customer

import "net/http"

const CustomersPath = "/api/v1/customers"
const CustomerDetailsPath = CustomersPath + "/"

func ProtectedRoutes(handler *Handler) map[string]http.Handler {
	return map[string]http.Handler{
		CustomersPath:       http.HandlerFunc(handler.Customers),
		CustomerDetailsPath: http.HandlerFunc(handler.CustomerDetails),
	}
}
