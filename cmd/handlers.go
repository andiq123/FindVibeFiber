package main

import (
	"github.com/andiq123/FindVibeFiber/internals/core/services"
	"github.com/andiq123/FindVibeFiber/internals/handlers"
	"github.com/andiq123/FindVibeFiber/internals/repositories"
	"github.com/andiq123/FindVibeFiber/internals/scrapper"
	"gorm.io/gorm"
)

func getAllHandlers(db *gorm.DB) (*handlers.HealthHandlers, *handlers.AuthHandlers, *handlers.FavoritesHandlers, *handlers.SuggestionsHandlers, *handlers.MusicFinderHandlers) {
	healthHandlers := handlers.NewHealthHandlers()

	authRepository := repositories.NewAuthRepository(db)
	authService := services.NewAuthService(authRepository)
	authHandlers := handlers.NewAuthHandlers(authService)

	favoritesRepository := repositories.NewFavoritesRepository(db)
	favoritesService := services.NewFavoritesService(favoritesRepository, authRepository)
	favoritesHandlers := handlers.NewFavoritesHandlers(favoritesService)

	suggestionsService := services.NewSuggestionsService()
	suggestionsHandlers := handlers.NewSuggestionsHandlers(suggestionsService)

	colly := scrapper.GetInstance()
	musicFinderService := services.NewMusicFinderService(colly)
	musicFinderHandlers := handlers.NewMusicFinderHandlers(musicFinderService)

	return healthHandlers, authHandlers, favoritesHandlers, suggestionsHandlers, musicFinderHandlers
}
