package handlers

import (
	"encoding/json"
	"fmt"
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

	image, err := ch.itunesCover(c, q)
	if err != nil {
		// ponytail: optional fill — empty art beats 500 when Apple flakes
		return c.JSON(fiber.Map{"image": ""})
	}
	return c.JSON(fiber.Map{"image": image})
}

func (ch *CoverHandler) itunesCover(c fiber.Ctx, q string) (string, error) {
	apiURL := "https://itunes.apple.com/search?media=music&entity=song&limit=1&term=" + url.QueryEscape(q)
	req, err := http.NewRequestWithContext(c.Context(), http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := ch.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("itunes %d", resp.StatusCode)
	}

	var data struct {
		Results []struct {
			ArtworkURL100 string `json:"artworkUrl100"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(data.Results) == 0 || data.Results[0].ArtworkURL100 == "" {
		return "", nil
	}
	return upgradeItunesArtwork(data.Results[0].ArtworkURL100), nil
}

func upgradeItunesArtwork(u string) string {
	return strings.Replace(u, "100x100", "1000x1000", 1)
}
