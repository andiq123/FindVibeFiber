package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
)

type CoverHandler struct {
	client *http.Client
}

func NewCoverHandler(client *http.Client) *CoverHandler {
	return &CoverHandler{client: client}
}

// GET /cover?q=artist+title → {"image":"https://..."} or {"image":""}.
func (ch *CoverHandler) GetCover(c fiber.Ctx) error {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "query parameter 'q' is required"})
	}
	if err := utils.ValidateQuery(q); err != nil {
		return HandleError(c, err)
	}

	// ponytail: optional fill — empty art beats 500 when Apple flakes
	return c.JSON(fiber.Map{"image": FetchItunesCover(c.Context(), ch.client, q)})
}

// FetchItunesCover returns a large artwork URL, or "" on miss/error.
func FetchItunesCover(ctx context.Context, client *http.Client, q string) string {
	q = strings.TrimSpace(q)
	if q == "" || client == nil {
		return ""
	}
	apiURL := "https://itunes.apple.com/search?media=music&entity=song&limit=1&term=" + url.QueryEscape(q)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var data struct {
		Results []struct {
			ArtworkURL100 string `json:"artworkUrl100"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	if len(data.Results) == 0 || data.Results[0].ArtworkURL100 == "" {
		return ""
	}
	return strings.Replace(data.Results[0].ArtworkURL100, "100x100", "600x600", 1)
}
