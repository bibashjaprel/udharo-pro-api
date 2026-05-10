package auth

import (
	"net/http"
)

type Middleware = func(http.Handler) http.Handler

func RegisterRoutes(mux *http.ServeMux, handler *Handler, authMiddleware Middleware) {
	mux.HandleFunc("/api/v1/auth/signup", handler.Signup)
	mux.HandleFunc("/api/v1/auth/login", handler.Login)
	mux.Handle("/api/v1/auth/logout", authMiddleware(http.HandlerFunc(handler.Logout)))
}
