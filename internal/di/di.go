package di

import (
	"github.com/andiq123/FindVibeFiber/internal/api"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/core/services"
	"github.com/andiq123/FindVibeFiber/internal/core/services/providers"
	"github.com/andiq123/FindVibeFiber/internal/persistence"
	"github.com/andiq123/FindVibeFiber/internal/utils"
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

	httpClient := utils.GetHTTPClient()
	musicProviders := []ports.IMusicProvider{
		providers.NewMuzskyProvider(httpClient),
		providers.NewMuzVibeProvider(httpClient),
		providers.NewMuzikaVsemProvider(httpClient),
	}
	searchConfig := domain.DefaultSearchConfig()
	musicAggregatorService := services.NewMusicAggregatorService(musicProviders, searchConfig)
	musicFinderHandler := api.NewMusicFinderHandler(musicAggregatorService)

	return healthHandler, authHandler, favoritesHandler, suggestionsHandler, musicFinderHandler
}
