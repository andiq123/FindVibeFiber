package main

import (
	"os"

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
	db := database.InitDb()
	defer database.CloseDb(db)

	healthHandler, authHandler, favoritesHandler, suggestionsHandler, searchHandler := di.InitializeHandlers(db)

	srv := server.NewServer(healthHandler, authHandler, favoritesHandler, suggestionsHandler, searchHandler)
	srv.Initialize()
	srv.Start()
}
