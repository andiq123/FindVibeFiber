package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/core/services"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
)

// Match explore: don't let cold iTunes dominate search p95.
const searchCoverBudget = 2500 * time.Millisecond

type SearchHandler struct {
	searchService ports.ISearchService
	covers        *services.CoverService
}

func NewSearchHandler(
	searchService ports.ISearchService,
	covers *services.CoverService,
) *SearchHandler {
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

	// Fill art with a short budget — empty image beats a slow search response.
	coverCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Context()), searchCoverBudget)
	sh.covers.FillSongs(coverCtx, response.Songs)
	cancel()

	// Align with SearchService TTL (2m); clients may revalidate sooner.
	c.Set("Cache-Control", "private, max-age=120")
	return c.JSON(response)
}
