package main

import (
	"os"

	"github.com/andiq123/FindVibeFiber/internal/config"
	"github.com/andiq123/FindVibeFiber/internal/database"
	"github.com/andiq123/FindVibeFiber/internal/di"
	"github.com/andiq123/FindVibeFiber/internal/server"
	"github.com/joho/godotenv"
)

func init() {
	if os.Getenv("IS_LOCAL") == "true" {
		_ = godotenv.Load()
	}
}

func main() {
	cfg := config.LoadConfig()
	db := database.InitDb(cfg.Database)
	defer database.CloseDb(db)

	srv := server.NewServer(cfg.Server, di.InitializeHandlers(db, cfg))
	srv.Start()
}
