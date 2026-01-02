package providers

import (
	"net/http"
	"slices"
	"strings"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/utils"
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
	normalizedQuery := utils.NormalizeString(query)
	normalizedTitle := utils.NormalizeString(song.Title)
	normalizedArtist := utils.NormalizeString(song.Artist)

	if normalizedTitle == normalizedQuery {
		return 1.0
	}

	if normalizedArtist == normalizedQuery {
		return 0.9
	}

	titleContains := strings.Contains(normalizedTitle, normalizedQuery)
	if titleContains {
		return 0.8
	}

	if strings.Contains(normalizedArtist, normalizedQuery) {
		return 0.7
	}

	queryWords := strings.Fields(normalizedQuery)
	if len(queryWords) == 0 {
		return 0.5
	}

	titleWords := strings.Fields(normalizedTitle)
	matches := 0
	for _, qw := range queryWords {
		if slices.Contains(titleWords, qw) {
			matches++
		}
	}

	return float64(matches) / float64(len(queryWords)) * 0.6
}

func (bp *BaseProvider) AddBrowserHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}
