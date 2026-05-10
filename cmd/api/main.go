package main

import (
	"log"

	"github.com/bibashjaprel/udharo-pro-api/internal/config"
	"github.com/bibashjaprel/udharo-pro-api/internal/database"
	"github.com/bibashjaprel/udharo-pro-api/internal/server"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	srv := server.New(cfg, db)
	log.Printf("Udharo Pro API running on %s", srv.Addr())

	if err := srv.HTTPServer().ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
