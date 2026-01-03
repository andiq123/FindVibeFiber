package handlers

import (
	"net/http"
	"strconv"

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

	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	response, err := sh.searchService.Search(c.Context(), query, page)
	if err != nil {
		return HandleError(c, err)
	}

	if len(response.Songs) == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no songs found"})
	}

	return c.JSON(response)
}
