package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/core/services"
	"github.com/andiq123/FindVibeFiber/internal/core/services/providers"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
)

type SearchHandler struct {
	searchService ports.ISearchService
	covers        *services.CoverService
	mp3musics     *providers.Mp3musicsProvider
}

func NewSearchHandler(
	searchService ports.ISearchService,
	covers *services.CoverService,
	mp3musics *providers.Mp3musicsProvider,
) *SearchHandler {
	return &SearchHandler{searchService: searchService, covers: covers, mp3musics: mp3musics}
}

// GET /resolve/mp3musics?file=/file/{hex}|{hex} → { link: "https://…audio…" }
func (sh *SearchHandler) ResolveMp3musics(c fiber.Ctx) error {
	if sh.mp3musics == nil {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": "Mp3musics unavailable"})
	}
	file := strings.TrimSpace(c.Query("file"))
	if file == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Context()), 20*time.Second)
	defer cancel()
	link, err := sh.mp3musics.ResolveFile(ctx, file)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{"error": "Couldn't resolve Mp3musics stream"})
	}
	return c.JSON(fiber.Map{"link": link})
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
