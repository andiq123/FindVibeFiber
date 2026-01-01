package providers

import (
	"net/http"
	"strings"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

type BaseProvider struct {
	name     string
	priority int
	client   *http.Client
}

func NewBaseProvider(name string, priority int, client *http.Client) *BaseProvider {
	return &BaseProvider{
		name:     name,
		priority: priority,
		client:   client,
	}
}

func (bp *BaseProvider) Name() string {
	return bp.name
}

func (bp *BaseProvider) Priority() int {
	return bp.priority
}

func (bp *BaseProvider) GetClient() *http.Client {
	return bp.client
}

func (bp *BaseProvider) CalculateBasicMatchScore(song domain.Song, query string) float64 {
	normalizedQuery := normalizeString(query)
	normalizedTitle := normalizeString(song.Title)
	normalizedArtist := normalizeString(song.Artist)

	if normalizedTitle == normalizedQuery {
		return 1.0
	}

	if normalizedArtist == normalizedQuery {
		return 0.9
	}

	if strings.Contains(normalizedTitle, normalizedQuery) {
		return 0.8
	}

	if strings.Contains(normalizedArtist, normalizedQuery) {
		return 0.7
	}

	queryWords := strings.Fields(normalizedQuery)
	titleWords := strings.Fields(normalizedTitle)

	matches := 0
	for _, qw := range queryWords {
		for _, tw := range titleWords {
			if qw == tw {
				matches++
				break
			}
		}
	}

	if len(queryWords) > 0 {
		return float64(matches) / float64(len(queryWords)) * 0.6
	}

	return 0.5
}

func (bp *BaseProvider) AddBrowserHeaders(req *http.Request, referer string) {
	headers := map[string]string{
		"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "en-US,en;q=0.9",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Ch-Ua":                 `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`,
		"Sec-Ch-Ua-Mobile":          "?0",
		"Sec-Ch-Ua-Platform":        `"macOS"`,
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "same-origin",
		"Sec-Fetch-User":            "?1",
	}

	if referer != "" {
		headers["Referer"] = referer
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}
