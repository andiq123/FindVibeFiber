package handlers

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/gofiber/fiber/v3"
)

type SuggestionsHandler struct {
	suggestionsService ports.ISuggestionsService
}

func NewSuggestionsHandler(suggestionsService ports.ISuggestionsService) *SuggestionsHandler {
	return &SuggestionsHandler{
		suggestionsService: suggestionsService,
	}
}

func (sh *SuggestionsHandler) GetSuggestions(c fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "query parameter 'q' is required"})
	}

	suggestions, err := sh.suggestionsService.GetSuggestions(c.Context(), query)
	if err != nil {
		return HandleError(c, err)
	}

	return c.JSON(suggestions)
}
