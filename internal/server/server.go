package server

import (
	"fmt"
	"log"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
)

type Server struct {
	healthHandler      ports.IHealthHandler
	authHandler        ports.IAuthHandler
	favoritesHandler   ports.IFavoritesHandler
	suggestionsHandler ports.ISuggestionsHandler
	musicFinderHandler ports.IMusicFinderHandler
}

func NewServer(
	healthHandler ports.IHealthHandler,
	authHandler ports.IAuthHandler,
	favoritesHandler ports.IFavoritesHandler,
	suggestionsHandler ports.ISuggestionsHandler,
	musicFinderHandler ports.IMusicFinderHandler) *Server {
	return &Server{
		healthHandler:      healthHandler,
		authHandler:        authHandler,
		suggestionsHandler: suggestionsHandler,
		musicFinderHandler: musicFinderHandler,
		favoritesHandler:   favoritesHandler,
	}
}

func (s *Server) Initialize() {
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://find-vibe.vercel.app", "http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "ngrok-skip-browser-warning"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"Content-Length"},
	}))
	app.Use(recoverer.New())

	app.Get("/health", s.healthHandler.GetHealth)

	suggestRoutes := app.Group("/suggest")
	suggestRoutes.Get("/", s.suggestionsHandler.GetSuggestions)

	searchRoutes := app.Group("/search")
	searchRoutes.Get("/", s.musicFinderHandler.FindMusic)

	favoritesRoutes := app.Group("/favorites")
	favoritesRoutes.Get("/:userId", s.favoritesHandler.GetFavorites)
	favoritesRoutes.Post("/:userId", s.favoritesHandler.AddFavorite)
	favoritesRoutes.Delete("/:songId", s.favoritesHandler.DeleteFavorite)
	favoritesRoutes.Put("/", s.favoritesHandler.ReorderFavorites)

	app.Get("/:username", s.authHandler.AuthenticateUser)

	port := utils.GetEnvOrDef("PORT", "8080")

	log.Println("Server listening on port: ", port)
	log.Fatal(app.Listen(fmt.Sprintf(":%v", port), fiber.ListenConfig{EnablePrefork: !utils.IsDebug()}))
}
