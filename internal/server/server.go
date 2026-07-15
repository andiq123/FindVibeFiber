package server

import (
	"fmt"
	"strings"
	"sync/atomic"
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
	h   atomic.Pointer[di.Handlers]
}

// NewServer listens with /health immediately; call Mount after DB is ready.
func NewServer(cfg config.ServerConfig) *Server {
	s := &Server{cfg: cfg}
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

	// Always answer — Render free tier 504s until something listens.
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON("Pong")
	})
	app.Get("/health/sources", s.withHandlers(func(h *di.Handlers, c fiber.Ctx) error {
		return h.Health.GetSources(c)
	}))
	app.Get("/suggest", s.withHandlers(func(h *di.Handlers, c fiber.Ctx) error {
		return h.Suggestions.GetSuggestions(c)
	}))
	app.Get("/search", s.withHandlers(func(h *di.Handlers, c fiber.Ctx) error {
		return h.Search.Search(c)
	}))

	favorites := app.Group("/favorites")
	favorites.Get("/:userId", s.withHandlers(func(h *di.Handlers, c fiber.Ctx) error {
		return h.Favorites.GetFavorites(c)
	}))
	favorites.Post("/:userId", s.withHandlers(func(h *di.Handlers, c fiber.Ctx) error {
		return h.Favorites.AddFavorite(c)
	}))
	favorites.Delete("/:songId", s.withHandlers(func(h *di.Handlers, c fiber.Ctx) error {
		return h.Favorites.DeleteFavorite(c)
	}))
	favorites.Put("/", s.withHandlers(func(h *di.Handlers, c fiber.Ctx) error {
		return h.Favorites.ReorderFavorites(c)
	}))

	app.Get("/:username", s.withHandlers(func(h *di.Handlers, c fiber.Ctx) error {
		return h.Auth.AuthenticateUser(c)
	}))

	s.app = app
	return s
}

func (s *Server) Mount(h di.Handlers) {
	s.h.Store(&h)
	utils.GetLogger().Info("API handlers mounted")
}

func (s *Server) withHandlers(fn func(*di.Handlers, fiber.Ctx) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		h := s.h.Load()
		if h == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "starting"})
		}
		return fn(h, c)
	}
}

func (s *Server) Start() {
	utils.GetLogger().Info("Server starting", "port", s.cfg.Port)
	if err := s.app.Listen(fmt.Sprintf(":%s", s.cfg.Port)); err != nil {
		utils.GetLogger().Error("Server failed to start", "error", err)
		panic(err)
	}
}
