package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/core/services"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
)

// Match explore: don't let cold iTunes dominate search p95.
const searchCoverBudget = 2500 * time.Millisecond
const searchSongCoverBudget = 400 * time.Millisecond

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

type searchStreamEvent struct {
	Type       string                 `json:"type"`
	Artists    []domain.SearchArtist  `json:"artists,omitempty"`
	Albums     []domain.ArtistAlbum   `json:"albums,omitempty"`
	Pagination *domain.PaginationInfo `json:"pagination,omitempty"`
	Song       *domain.Song           `json:"song,omitempty"`
	Songs      []domain.Song          `json:"songs,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// GET /search?q=&page= → JSON SearchResponse.
// GET /search?q=&stream=1 → NDJSON: meta → song|songs* → done (mapped hits as ready).
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

	stream := c.Query("stream") == "1" || strings.EqualFold(c.Query("stream"), "true")
	if stream {
		return sh.streamSearch(c, query, page)
	}

	response, err := sh.searchService.Search(c.Context(), query, page)
	if err != nil {
		return HandleError(c, err)
	}

	if len(response.Songs) == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "no songs found"})
	}

	// Last.fm discovers the catalog; providers only map playable stream URLs.
	// Fill art with a short budget — empty image beats a slow search response.
	coverCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Context()), searchCoverBudget)
	sh.covers.FillSongs(coverCtx, response.Songs)
	sh.covers.FillArtists(coverCtx, response.Artists)
	sh.covers.FillAlbums(coverCtx, response.Albums)
	cancel()

	// Align with SearchService TTL (2m); clients may revalidate sooner.
	c.Set("Cache-Control", "private, max-age=120")
	return c.JSON(response)
}

func (sh *SearchHandler) streamSearch(c fiber.Ctx, query string, page int) error {
	c.Set("Content-Type", "application/x-ndjson")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		enc := json.NewEncoder(w)
		write := func(ev searchStreamEvent) bool {
			if err := enc.Encode(ev); err != nil {
				return false
			}
			return w.Flush() == nil
		}

		var streamed int
		var buf []domain.Song
		flushSongs := func() bool {
			if len(buf) == 0 {
				return true
			}
			batch := buf
			buf = nil
			if len(batch) == 1 {
				if !write(searchStreamEvent{Type: "song", Song: &batch[0]}) {
					return false
				}
			} else if !write(searchStreamEvent{Type: "songs", Songs: batch}) {
				return false
			}
			streamed += len(batch)
			return true
		}

		onMeta := func(p domain.SearchProgress) error {
			artists := append([]domain.SearchArtist(nil), p.Artists...)
			albums := append([]domain.ArtistAlbum(nil), p.Albums...)
			coverCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Context()), searchCoverBudget)
			sh.covers.FillArtists(coverCtx, artists)
			sh.covers.FillAlbums(coverCtx, albums)
			cancel()
			if !write(searchStreamEvent{
				Type:       "meta",
				Artists:    artists,
				Albums:     albums,
				Pagination: p.Pagination,
			}) {
				return context.Canceled
			}
			return nil
		}
		onSong := func(song domain.Song) error {
			songs := []domain.Song{song}
			// Skip cover wait when Last.fm already attached art — keeps the stream snappy.
			if strings.TrimSpace(songs[0].Image) == "" || strings.Contains(songs[0].Image, "2a96cbd8b46e442fc41c2b86b821562f") {
				coverCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Context()), searchSongCoverBudget)
				sh.covers.FillSongs(coverCtx, songs)
				cancel()
			}
			buf = append(buf, songs[0])
			// First hit flushes immediately; later hits batch up to 5 for fewer UI rebuilds.
			if streamed == 0 || len(buf) >= 5 {
				if !flushSongs() {
					return context.Canceled
				}
			}
			return nil
		}

		resp, err := sh.searchService.SearchWithProgress(c.Context(), query, page, onMeta, onSong)
		if !flushSongs() {
			return
		}
		if err != nil && streamed == 0 {
			_ = write(searchStreamEvent{Type: "error", Error: "Couldn't search"})
			return
		}
		if (resp == nil || len(resp.Songs) == 0) && streamed == 0 {
			_ = write(searchStreamEvent{Type: "error", Error: "no songs found"})
			return
		}
		_ = write(searchStreamEvent{Type: "done"})
	})
}
