package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/config"
	"github.com/andiq123/FindVibeFiber/internal/core/constants"
	"github.com/andiq123/FindVibeFiber/internal/di"
	"github.com/andiq123/FindVibeFiber/internal/middleware"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type Server struct {
	app *fiber.App
	cfg config.ServerConfig
}

func NewServer(cfg config.ServerConfig, h di.Handlers) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		BodyLimit:    constants.MaxRequestSize,
		// Render sits behind a proxy; trust private/link-local so c.IP() is the client
		TrustProxy:       true,
		ProxyHeader:      fiber.HeaderXForwardedFor,
		TrustProxyConfig: fiber.TrustProxyConfig{Private: true, Loopback: true, LinkLocal: true},
	})

	app.Use(cors.New(middleware.NewCORS()))
	app.Use(recover.New())
	app.Use(compress.New())
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: time.Minute,
		Next: func(c fiber.Ctx) bool {
			return c.Path() == "/health" || strings.HasPrefix(c.Path(), "/health/")
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many requests"})
		},
	}))

	app.Get("/health", h.Health.GetHealth)
	app.Get("/health/sources", h.Health.GetSources)
	app.Get("/suggest", h.Suggestions.GetSuggestions)
	app.Get("/search", h.Search.Search)

	favorites := app.Group("/favorites")
	favorites.Get("/:userId", h.Favorites.GetFavorites)
	favorites.Post("/:userId", h.Favorites.AddFavorite)
	favorites.Delete("/:songId", h.Favorites.DeleteFavorite)
	favorites.Put("/", h.Favorites.ReorderFavorites)

	app.Get("/:username", h.Auth.AuthenticateUser)

	return &Server{app: app, cfg: cfg}
}

func (s *Server) Start() {
	utils.GetLogger().Info("Server starting", "port", s.cfg.Port)
	if err := s.app.Listen(fmt.Sprintf(":%s", s.cfg.Port)); err != nil {
		utils.GetLogger().Error("Server failed to start", "error", err)
		panic(err)
	}
}
