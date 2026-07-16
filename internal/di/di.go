package di

import (
	"os"

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

type Handlers struct {
	Health      *handlers.HealthHandler
	Auth        *handlers.AuthHandler
	Favorites   *handlers.FavoritesHandler
	Suggestions *handlers.SuggestionsHandler
	Search      *handlers.SearchHandler
	Cover       *handlers.CoverHandler
	Recommend   *handlers.RecommendHandler
	Lyrics      *handlers.LyricsHandler
	Spotify     *handlers.SpotifyHandler
}

func InitializeHandlers(db *gorm.DB, cfg *config.AppConfig) Handlers {
	authRepository := repository.NewAuthRepository(db)
	favoritesRepository := repository.NewFavoritesRepository(db)

	httpClient := utils.NewHTTPClient(
		cfg.HTTP.Timeout,
		cfg.HTTP.MaxIdleConns,
		cfg.HTTP.MaxIdlePerHost,
		cfg.HTTP.IdleTimeout,
	)

	searchConfig := domain.DefaultSearchConfig()
	searchConfig.MaxResults = cfg.Search.MaxResults

	covers := services.NewCoverService(httpClient)
	searchSvc := services.NewSearchService(
		[]ports.IMusicProvider{
			providers.NewMuzJamProvider(httpClient),
			providers.NewMp3mnProvider(httpClient),
		},
		searchConfig,
		cfg.Search.Timeout,
	)

	return Handlers{
		Health:      handlers.NewHealthHandler(httpClient),
		Auth:        handlers.NewAuthHandler(services.NewAuthService(authRepository)),
		Favorites:   handlers.NewFavoritesHandler(services.NewFavoritesService(favoritesRepository, authRepository)),
		Suggestions: handlers.NewSuggestionsHandler(services.NewSuggestionsService(httpClient)),
		Cover:       handlers.NewCoverHandler(covers),
		Search:      handlers.NewSearchHandler(searchSvc, covers),
		Recommend:   handlers.NewRecommendHandler(httpClient, os.Getenv("LASTFM_API_KEY"), searchSvc, covers),
		Lyrics:      handlers.NewLyricsHandler(httpClient),
		Spotify:     handlers.NewSpotifyHandler(httpClient),
	}
}
