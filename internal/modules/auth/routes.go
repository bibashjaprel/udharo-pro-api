package auth

import (
	"net/http"
)

const (
	SignupPath = "/api/v1/auth/signup"
	LoginPath  = "/api/v1/auth/login"
	LogoutPath = "/api/v1/auth/logout"
	MePath     = "/api/v1/auth/me"
)

func RegisterPublicRoutes(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc(SignupPath, handler.Signup)
	mux.HandleFunc(LoginPath, handler.Login)
}

func ProtectedRoutes(handler *Handler) map[string]http.Handler {
	return map[string]http.Handler{
		LogoutPath: http.HandlerFunc(handler.Logout),
		MePath:     http.HandlerFunc(handler.Me),
	}
}
