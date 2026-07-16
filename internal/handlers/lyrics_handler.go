package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
)

var lrcStamp = regexp.MustCompile(`(?m)^\[[0-9.:]+\]\s?`)

type LyricsHandler struct {
	client *http.Client
}

func NewLyricsHandler(client *http.Client) *LyricsHandler {
	return &LyricsHandler{client: client}
}

// GET /lyrics?artist=&title= → {"lyrics":"..."} or {"error","code"}.
func (h *LyricsHandler) GetLyrics(c fiber.Ctx) error {
	artist := strings.TrimSpace(c.Query("artist"))
	title := strings.TrimSpace(c.Query("title"))
	if artist == "" || title == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Artist and title are required",
			"code":  "bad_request",
		})
	}

	text, code, err := h.lookup(c.Context(), artist, title)
	if err != nil {
		return c.Status(http.StatusBadGateway).JSON(fiber.Map{
			"error": "Couldn't reach lyrics service",
			"code":  "upstream",
		})
	}
	switch code {
	case "instrumental":
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "This track is instrumental",
			"code":  code,
		})
	case "not_found":
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "No lyrics for this track",
			"code":  code,
		})
	}
	return c.JSON(fiber.Map{"lyrics": text})
}

type lrclibHit struct {
	PlainLyrics  string `json:"plainLyrics"`
	SyncedLyrics string `json:"syncedLyrics"`
	Instrumental bool   `json:"instrumental"`
}

func (h *LyricsHandler) lookup(ctx context.Context, artist, title string) (text, code string, err error) {
	// ponytail: /search needs no album/duration — /get does.
	hits, err := h.search(ctx, url.Values{
		"artist_name": {artist},
		"track_name":  {title},
	})
	if err != nil {
		return "", "", err
	}
	if text = pickLyrics(hits); text != "" {
		return text, "", nil
	}
	hits, err = h.search(ctx, url.Values{"q": {artist + " " + title}})
	if err != nil {
		return "", "", err
	}
	if text = pickLyrics(hits); text != "" {
		return text, "", nil
	}
	if allInstrumental(hits) {
		return "", "instrumental", nil
	}
	return "", "not_found", nil
}

func allInstrumental(hits []lrclibHit) bool {
	if len(hits) == 0 {
		return false
	}
	for _, h := range hits {
		if !h.Instrumental {
			return false
		}
	}
	return true
}

func (h *LyricsHandler) search(ctx context.Context, q url.Values) ([]lrclibHit, error) {
	u := "https://lrclib.net/api/search?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "FindVibe/1.0 (https://github.com/andiq123/FindVibeFiber)")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var hits []lrclibHit
	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		return nil, err
	}
	return hits, nil
}

func pickLyrics(hits []lrclibHit) string {
	for _, hit := range hits {
		if hit.Instrumental {
			continue
		}
		if t := strings.TrimSpace(hit.PlainLyrics); t != "" {
			return t
		}
	}
	for _, hit := range hits {
		if hit.Instrumental {
			continue
		}
		if t := strings.TrimSpace(hit.SyncedLyrics); t != "" {
			return strings.TrimSpace(lrcStamp.ReplaceAllString(t, ""))
		}
	}
	return ""
}
