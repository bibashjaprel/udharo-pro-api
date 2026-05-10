package server

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bibashjaprel/udharo-pro-api/internal/config"
	"github.com/bibashjaprel/udharo-pro-api/internal/middleware"
	"github.com/bibashjaprel/udharo-pro-api/internal/modules/auth"
	"github.com/bibashjaprel/udharo-pro-api/internal/modules/shop"
)

type Server struct {
	config config.Config
	db     *pgxpool.Pool
	mux    *http.ServeMux
}

func New(cfg config.Config, db *pgxpool.Pool) *Server {
	s := &Server{
		config: cfg,
		db:     db,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) Addr() string {
	return ":" + s.config.AppPort
}

func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:    s.Addr(),
		Handler: s.Handler(),
	}
}

func (s *Server) authModule() (*auth.Handler, middleware.Middleware) {
	authService := auth.NewService(s.db, s.config.JWTSecret)
	authHandler := auth.NewHandler(authService)
	authMiddleware := middleware.Auth(s.config.JWTSecret, authService)

	return authHandler, authMiddleware
}

func (s *Server) shopModule() *shop.Handler {
	shopService := shop.NewService(s.db)
	return shop.NewHandler(shopService)
}
