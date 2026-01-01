package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type MuzskyProvider struct {
	*BaseProvider
	sourceURL string
}

var _ ports.IMusicProvider = (*MuzskyProvider)(nil)

func NewMuzskyProvider(client *http.Client) *MuzskyProvider {
	return &MuzskyProvider{
		BaseProvider: NewBaseProvider("Muzsky", 9, client),
		sourceURL:    "https://muzsky.net/search/",
	}
}

func (mp *MuzskyProvider) Search(ctx context.Context, query string) ([]domain.ProviderResult, error) {
	req, err := mp.createSearchRequest(ctx, query)
	if err != nil {
		return nil, err
	}

	resp, err := mp.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("muzsky: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("muzsky: unexpected status code: %d", resp.StatusCode)
	}

	return mp.parseResults(resp.Body, query)
}

func (mp *MuzskyProvider) createSearchRequest(ctx context.Context, query string) (*http.Request, error) {
	apiURL := mp.sourceURL + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("muzsky: request creation failed: %w", err)
	}

	mp.AddBrowserHeaders(req, "https://muzsky.net/")
	return req, nil
}

func (mp *MuzskyProvider) parseResults(body io.Reader, query string) ([]domain.ProviderResult, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("muzsky: failed to parse HTML: %w", err)
	}

	var results []domain.ProviderResult
	rank := 1

	doc.Find("table.table-striped tbody tr").Each(func(_ int, s *goquery.Selection) {
		// Extract image from data-src attribute (lazy loaded)
		image := s.Find("img.lazy").AttrOr("data-src", "")

		// Extract title and artist from link text
		linkText := strings.TrimSpace(s.Find("span.tablestyle.tablecolor a").Text())
		if linkText == "" {
			return
		}

		// Parse artist and title from format "Artist - Title"
		parts := strings.SplitN(linkText, " - ", 2)
		var title, artist string

		if len(parts) == 2 {
			artist = strings.TrimSpace(parts[0])
			title = strings.TrimSpace(parts[1])
		} else {
			// If no dash separator, treat entire text as title
			title = linkText
			artist = "Unknown"
		}

		// Extract download link from data-id attribute which contains the direct link
		downloadLink := s.Find("div.list-songs").AttrOr("data-id", "")
		// Ensure the link ends with a backslash as per user requirement
		if !strings.HasSuffix(downloadLink, "\\") {
			downloadLink += "\\"
		}

		if title != "" {
			song := domain.NewSong(title, artist, image, downloadLink)
			matchScore := mp.CalculateBasicMatchScore(*song, query)

			results = append(results, *domain.NewProviderResult(
				*song,
				mp.Name(),
				matchScore,
				rank,
			))
			rank++
		}
	})

	return results, nil
}
