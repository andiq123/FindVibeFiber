package ports

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

type ISuggestionsService interface {
	GetSuggestions(ctx context.Context, query string) ([]string, error)
}

type ISuggestionsHandler interface {
	GetSuggestions(c fiber.Ctx) error
}
