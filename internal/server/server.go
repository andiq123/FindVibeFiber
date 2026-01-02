package server

import (
	"fmt"
	"log"

	"github.com/andiq123/FindVibeFiber/internal/handlers"
	"github.com/andiq123/FindVibeFiber/internal/middleware"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
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
	s.app = fiber.New()
	s.setupMiddleware()
	s.setupRoutes()
}

func (s *Server) setupMiddleware() {
	s.app.Use(cors.New(middleware.NewCORS()))
	s.app.Use(recover.New())
}

func (s *Server) setupRoutes() {
	s.app.Get("/health", s.healthHandler.GetHealth)

	s.app.Group("/suggest").Get("/", s.suggestionsHandler.GetSuggestions)
	s.app.Group("/search").Get("/", s.searchHandler.Search)

	favorites := s.app.Group("/favorites")
	favorites.Get("/:userId", s.favoritesHandler.GetFavorites)
	favorites.Post("/:userId", s.favoritesHandler.AddFavorite)
	favorites.Delete("/:songId", s.favoritesHandler.DeleteFavorite)
	favorites.Put("/", s.favoritesHandler.ReorderFavorites)

	s.app.Get("/:username", s.authHandler.AuthenticateUser)
}

func (s *Server) Start() {
	port := utils.GetEnvOrDef("PORT", "8080")
	log.Printf("Server listening on port %s", port)

	if err := s.app.Listen(fmt.Sprintf(":%s", port), fiber.ListenConfig{
		EnablePrefork: !utils.IsDebug(),
	}); err != nil {
		log.Fatal(err)
	}
}
