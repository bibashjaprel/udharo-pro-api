package server

import (
	"net/http"

	"github.com/bibashjaprel/udharo-pro-api/internal/middleware"
	"github.com/bibashjaprel/udharo-pro-api/internal/modules/auth"
	"github.com/bibashjaprel/udharo-pro-api/internal/modules/customer"
	"github.com/bibashjaprel/udharo-pro-api/internal/modules/ledger"
	"github.com/bibashjaprel/udharo-pro-api/internal/modules/shop"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.health)

	authHandler, authMiddleware := s.authModule()
	auth.RegisterPublicRoutes(s.mux, authHandler)
	s.registerProtectedRoutes(auth.ProtectedRoutes(authHandler), authMiddleware, middleware.Tenant())
	s.registerProtectedRoutes(shop.ProtectedRoutes(s.shopModule()), authMiddleware, middleware.Tenant())
	customerHandler := s.customerModule()
	ledgerHandler := s.ledgerModule()
	s.registerProtectedRoutes(customer.ProtectedRoutes(customerHandler), authMiddleware, middleware.Tenant())
	s.registerProtectedRoutes(map[string]http.Handler{
		customer.CustomerDetailsPath: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ledger.IsCreditPath(r.URL.Path) {
				ledgerHandler.CreateCreditEntry(w, r)
				return
			}
			if ledger.IsLedgerPath(r.URL.Path) {
				ledgerHandler.ListCustomerLedger(w, r)
				return
			}

			customerHandler.CustomerDetails(w, r)
		}),
	}, authMiddleware, middleware.Tenant())
}

func (s *Server) registerProtectedRoutes(routes map[string]http.Handler, middlewares ...middleware.Middleware) {
	for path, handler := range routes {
		s.mux.Handle(path, middleware.Chain(handler, middlewares...))
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed", "method not allowed")
		return
	}

	status := map[string]string{
		"status":   "ok",
		"service":  "udharo-pro-api",
		"database": "up",
	}

	if err := s.db.Ping(r.Context()); err != nil {
		status["status"] = "error"
		status["database"] = "down"
		response.JSON(w, http.StatusServiceUnavailable, status)
		return
	}

	response.JSON(w, http.StatusOK, status)
}
