package server

import (
	"fmt"
	"log"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/config"
	"github.com/andiq123/FindVibeFiber/internal/handlers"
	"github.com/andiq123/FindVibeFiber/internal/middleware"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type Server struct {
	app                *fiber.App
	healthHandler      *handlers.HealthHandler
	authHandler        *handlers.AuthHandler
	favoritesHandler   *handlers.FavoritesHandler
	suggestionsHandler *handlers.SuggestionsHandler
	searchHandler      *handlers.SearchHandler
}

func NewServer(
	healthHandler *handlers.HealthHandler,
	authHandler *handlers.AuthHandler,
	favoritesHandler *handlers.FavoritesHandler,
	suggestionsHandler *handlers.SuggestionsHandler,
	searchHandler *handlers.SearchHandler,
) *Server {
	return &Server{
		healthHandler:      healthHandler,
		authHandler:        authHandler,
		favoritesHandler:   favoritesHandler,
		suggestionsHandler: suggestionsHandler,
		searchHandler:      searchHandler,
	}
}

func (s *Server) Initialize() {
	appConfig := config.LoadConfig()
	s.app = fiber.New(fiber.Config{
		ReadTimeout:  appConfig.Server.ReadTimeout,
		WriteTimeout: appConfig.Server.WriteTimeout,
		IdleTimeout:  appConfig.Server.IdleTimeout,
	})

	s.app.Use(cors.New(middleware.NewCORS()))
	s.app.Use(recover.New())
	s.app.Use(compress.New())
	s.app.Use(middleware.RequestSizeLimit())

	s.app.Get("/health", s.healthHandler.GetHealth)
	s.app.Use(middleware.RateLimit(100, time.Minute))

	s.app.Get("/suggest", s.suggestionsHandler.GetSuggestions)
	s.app.Get("/search", s.searchHandler.Search)

	favorites := s.app.Group("/favorites")
	favorites.Get("/:userId", s.favoritesHandler.GetFavorites)
	favorites.Post("/:userId", s.favoritesHandler.AddFavorite)
	favorites.Delete("/:songId", s.favoritesHandler.DeleteFavorite)
	favorites.Put("/", s.favoritesHandler.ReorderFavorites)

	s.app.Get("/:username", s.authHandler.AuthenticateUser)
}

func (s *Server) Start() {
	appConfig := config.LoadConfig()
	port := appConfig.Server.Port
	utils.GetLogger().Info("Server starting", "port", port)

	if err := s.app.Listen(fmt.Sprintf(":%s", port)); err != nil {
		utils.GetLogger().Error("Server failed to start", "error", err)
		log.Fatal(err)
	}
}
