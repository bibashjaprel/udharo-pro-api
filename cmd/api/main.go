package main

import (
	"log"
	"net/http"

	"github.com/bibashjaprel/udharo-pro-api/internal/config"
	"github.com/bibashjaprel/udharo-pro-api/internal/database"
	"github.com/bibashjaprel/udharo-pro-api/internal/modules/auth"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	authService := auth.NewService(db, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)
	authMiddleware := auth.AuthMiddleware(cfg.JWTSecret, auth.NewSessionStore(db))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"error","service":"udharo-pro-api","database":"down"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"udharo-pro-api","database":"up"}`))
	})
	mux.HandleFunc("/api/v1/auth/signup", authHandler.Signup)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.Handle("/api/v1/auth/logout", authMiddleware(http.HandlerFunc(authHandler.Logout)))

	addr := ":" + cfg.AppPort
	log.Printf("Udharo Pro API running on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
