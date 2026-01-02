package handlers

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/gofiber/fiber/v3"
)

type SearchHandler struct {
	searchService ports.ISearchService
}

func NewSearchHandler(searchService ports.ISearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

func (sh *SearchHandler) Search(c fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "query parameter 'q' is required"})
	}

	songs, err := sh.searchService.Search(c.Context(), query)
	if err != nil {
		return HandleError(c, err)
	}

	if len(songs) == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no songs found"})
	}

	return c.JSON(songs)
}
