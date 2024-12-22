package server

import (
	"fmt"
	"log"

	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/andiq123/FindVibeFiber/internals/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
)

type Server struct {
	healthHandler       ports.IHealthHandlers
	authHandlers        ports.IAuthHandlers
	favoritesHandler    ports.IFavoritesHandler
	suggestionsHandlers ports.ISuggestionsHandlers
	musicFinderHandlers ports.IMusicFinderHandlers
}

func NewServer(
	healthHandler ports.IHealthHandlers,
	authHandlers ports.IAuthHandlers,
	favoritesHanlder ports.IFavoritesHandler,
	suggestionsHandlers ports.ISuggestionsHandlers,
	musicFinderHandlers ports.IMusicFinderHandlers) *Server {
	return &Server{
		healthHandler:       healthHandler,
		authHandlers:        authHandlers,
		suggestionsHandlers: suggestionsHandlers,
		musicFinderHandlers: musicFinderHandlers,
		favoritesHandler:    favoritesHanlder,
	}
}

func (s *Server) Initialize() {
	app := fiber.New()

	v1 := app.Group("/v1")

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"https://find-vibe.vercel.app", "http://localhost:4200"},
	}))
	app.Use(recoverer.New())

	v1.Get("/ping", s.healthHandler.GetHealth)

	authRoutes := v1.Group("/auth")
	authRoutes.Get("/:username", s.authHandlers.AuthenticateUser)

	suggestionsRoutes := v1.Group("/suggestions")
	suggestionsRoutes.Get("/", s.suggestionsHandlers.GetSuggestions)

	musicFinderRoutes := v1.Group("/music-finder")
	musicFinderRoutes.Get("/", s.musicFinderHandlers.FindMusic)

	favoritesRoutes := v1.Group("/favorites")
	favoritesRoutes.Get("/:userId", s.favoritesHandler.GetFavorites)
	favoritesRoutes.Post("/:userId", s.favoritesHandler.AddFavorite)
	favoritesRoutes.Delete("/:songId", s.favoritesHandler.DeleteFavorite)
	favoritesRoutes.Put("/", s.favoritesHandler.ReorderFavorites)

	port := utils.GetEnvOrDef("PORT", "8080")

	log.Println("Server listening on port: ", port)
	log.Fatal(app.Listen(fmt.Sprintf(":%v", port), fiber.ListenConfig{EnablePrefork: true}))
}
