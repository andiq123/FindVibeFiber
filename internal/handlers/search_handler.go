package handlers

import (
	"net/http"
	"strconv"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/core/services"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
)

type SearchHandler struct {
	searchService ports.ISearchService
	covers        *services.CoverService
}

func NewSearchHandler(searchService ports.ISearchService, covers *services.CoverService) *SearchHandler {
	return &SearchHandler{searchService: searchService, covers: covers}
}

func (sh *SearchHandler) Search(c fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "query parameter 'q' is required"})
	}

	if err := utils.ValidateQuery(query); err != nil {
		return HandleError(c, err)
	}

	page := 1
	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil {
			page = p
		}
	}

	if err := utils.ValidatePage(page); err != nil {
		return HandleError(c, err)
	}

	response, err := sh.searchService.Search(c.Context(), query, page)
	if err != nil {
		return HandleError(c, err)
	}

	if len(response.Songs) == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no songs found"})
	}

	// ponytail: fill only the HTTP response page — not every resolveOne inside radio/explore
	sh.covers.FillSongs(c.Context(), response.Songs)
	return c.JSON(response)
}
