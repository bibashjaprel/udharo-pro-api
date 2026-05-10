package server

import (
	"net/http"

	"github.com/bibashjaprel/udharo-pro-api/internal/modules/auth"
	"github.com/bibashjaprel/udharo-pro-api/internal/shared/response"
)

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.health)

	authHandler, authMiddleware := s.authModule()
	auth.RegisterRoutes(s.mux, authHandler, authMiddleware)
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
