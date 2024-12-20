package main

import (
	"github.com/andiq123/FindVibeFiber/internals/core/services"
	"github.com/andiq123/FindVibeFiber/internals/handlers"
	"github.com/andiq123/FindVibeFiber/internals/repositories"
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

	musicFinderService := services.NewMusicFinderService()
	musicFinderHandlers := handlers.NewMusicFinderHandlers(musicFinderService)

	return healthHandlers, authHandlers, favoritesHandlers, suggestionsHandlers, musicFinderHandlers
}
