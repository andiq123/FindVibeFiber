package main

import (
	"log"

	"github.com/andiq123/FindVibeFiber/internal/database"
	"github.com/andiq123/FindVibeFiber/internal/di"
	"github.com/andiq123/FindVibeFiber/internal/server"
	"github.com/andiq123/FindVibeFiber/internal/utils"
)

func init() {
	log.Println("Debug mode enabled: ", utils.IsDebug())
}

func main() {
	db := database.InitDb()
	defer database.CloseDb(db)

	healthHandler, authHandler, favoritesHandler, suggestionsHandler, searchHandler := di.InitializeHandlers(db)

	srv := server.NewServer(healthHandler, authHandler, favoritesHandler, suggestionsHandler, searchHandler)
	srv.Initialize()
	srv.Start()
}
