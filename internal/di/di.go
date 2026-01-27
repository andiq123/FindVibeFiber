package di

import (
	"github.com/andiq123/FindVibeFiber/internal/config"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/core/services"
	"github.com/andiq123/FindVibeFiber/internal/core/services/providers"
	"github.com/andiq123/FindVibeFiber/internal/handlers"
	"github.com/andiq123/FindVibeFiber/internal/repository"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"gorm.io/gorm"
)

func InitializeHandlers(db *gorm.DB) (*handlers.HealthHandler, *handlers.AuthHandler, *handlers.FavoritesHandler, *handlers.SuggestionsHandler, *handlers.SearchHandler) {
	healthHandler := handlers.NewHealthHandler()

	authRepository := repository.NewAuthRepository(db)
	authService := services.NewAuthService(authRepository)
	authHandler := handlers.NewAuthHandler(authService)

	favoritesRepository := repository.NewFavoritesRepository(db)
	favoritesService := services.NewFavoritesService(favoritesRepository, authRepository)
	favoritesHandler := handlers.NewFavoritesHandler(favoritesService)

	appConfig := config.LoadConfig()
	httpClient := utils.NewHTTPClient(
		appConfig.HTTP.Timeout,
		appConfig.HTTP.MaxIdleConns,
		appConfig.HTTP.MaxIdlePerHost,
		appConfig.HTTP.IdleTimeout,
	)
	suggestionsService := services.NewSuggestionsService(httpClient)
	suggestionsHandler := handlers.NewSuggestionsHandler(suggestionsService)
	musicProviders := []ports.IMusicProvider{
		providers.NewMuzVibeProvider(httpClient),
	}
	searchConfig := domain.DefaultSearchConfig()
	searchConfig.MaxResults = appConfig.Search.MaxResults
	searchService := services.NewSearchService(musicProviders, searchConfig, appConfig.Search.Timeout)
	searchHandler := handlers.NewSearchHandler(searchService)

	return healthHandler, authHandler, favoritesHandler, suggestionsHandler, searchHandler
}
