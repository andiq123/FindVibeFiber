package ports

import "github.com/gofiber/fiber/v3"

type ISuggestionsService interface {
	GetSuggestions(query string) ([]string, error)
}

type ISuggestionsHandlers interface {
	GetSuggestions(c fiber.Ctx) error
}
