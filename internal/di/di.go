package di

import (
	"github.com/andiq123/FindVibeFiber/internal/api"
	"github.com/andiq123/FindVibeFiber/internal/core/services"
	"github.com/andiq123/FindVibeFiber/internal/persistence"
	"gorm.io/gorm"
)

func InitializeHandlers(db *gorm.DB) (*api.HealthHandler, *api.AuthHandler, *api.FavoritesHandler, *api.SuggestionsHandler, *api.MusicFinderHandler) {
	healthHandler := api.NewHealthHandler()

	authRepository := persistence.NewAuthRepository(db)
	authService := services.NewAuthService(authRepository)
	authHandler := api.NewAuthHandler(authService)

	favoritesRepository := persistence.NewFavoritesRepository(db)
	favoritesService := services.NewFavoritesService(favoritesRepository, authRepository)
	favoritesHandler := api.NewFavoritesHandler(favoritesService)

	suggestionsService := services.NewSuggestionsService()
	suggestionsHandler := api.NewSuggestionsHandler(suggestionsService)

	musicFinderService := services.NewMusicFinderService()
	musicFinderHandler := api.NewMusicFinderHandler(musicFinderService)

	return healthHandler, authHandler, favoritesHandler, suggestionsHandler, musicFinderHandler
}
