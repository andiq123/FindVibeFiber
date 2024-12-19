package main

import (
	"github.com/andiq123/FindVibeFiber/internals/core/services"
	"github.com/andiq123/FindVibeFiber/internals/database"
	"github.com/andiq123/FindVibeFiber/internals/handlers"
	"github.com/andiq123/FindVibeFiber/internals/repositories"
	"github.com/andiq123/FindVibeFiber/internals/scrapper"
	"github.com/andiq123/FindVibeFiber/internals/server"
	"github.com/andiq123/FindVibeFiber/internals/utils"
)

func init() {
	utils.LoadEnv()
}

func main() {
	db := database.InitDb()

	//health
	healthHandlers := handlers.NewHealthHandlers()

	//auth
	authRepository := repositories.NewAuthRepository(db)
	authService := services.NewAuthService(authRepository)
	authHandlers := handlers.NewAuthHandlers(authService)

	//suggestions
	suggestionsService := services.NewSuggestionsService()
	suggestionsHandlers := handlers.NewSuggestionsHandlers(suggestionsService)

	//music finder
	collyCollector := scrapper.GetInstance()
	musicFinderService := services.NewMusicFinderService(collyCollector)
	musicFinderHandlers := handlers.NewMusicFinderHandlers(musicFinderService)

	//favorites
	favoritesRepository := repositories.NewFavoritesRepository(db)
	favoritesService := services.NewFavoritesService(favoritesRepository, authRepository)
	favoritesHandlers := handlers.NewFavoritesHandlers(favoritesService)

	httpServer := server.NewServer(healthHandlers, authHandlers, favoritesHandlers, suggestionsHandlers, musicFinderHandlers)
	httpServer.Initialize()
}
