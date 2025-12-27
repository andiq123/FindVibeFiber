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
	if utils.IsDebug() {
		err := utils.LoadEnv()
		if err != nil {
			log.Fatalf("Failed to load environment variables: %v", err)
		}
	}
}

func main() {
	db := database.InitDb()
	defer database.CloseDb(db)

	healthHandlers, authHandlers, favoritesHandlers, suggestionsHandlers, musicFinderHandlers := di.InitializeHandlers(db)

	httpServer := server.NewServer(healthHandlers, authHandlers, favoritesHandlers, suggestionsHandlers, musicFinderHandlers)
	httpServer.Initialize()
}
