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

type MuzVibeProvider struct {
	*BaseProvider
	sourceURL string
}

var _ ports.IMusicProvider = (*MuzVibeProvider)(nil)

func NewMuzVibeProvider(client *http.Client) *MuzVibeProvider {
	return &MuzVibeProvider{
		BaseProvider: NewBaseProvider("MuzVibe", 8, client),
		sourceURL:    "https://muzvibe.org/search/",
	}
}

func (mvp *MuzVibeProvider) Search(ctx context.Context, query string) ([]domain.ProviderResult, error) {
	req, err := mvp.createSearchRequest(ctx, query)
	if err != nil {
		return nil, err
	}

	resp, err := mvp.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("muzvibe: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("muzvibe: unexpected status code: %d", resp.StatusCode)
	}

	return mvp.parseResults(resp.Body, query)
}

func (mvp *MuzVibeProvider) createSearchRequest(ctx context.Context, query string) (*http.Request, error) {
	apiURL := mvp.sourceURL + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("muzvibe: request creation failed: %w", err)
	}

	mvp.AddBrowserHeaders(req, "https://muzvibe.org/")
	return req, nil
}

func (mvp *MuzVibeProvider) parseResults(body io.Reader, query string) ([]domain.ProviderResult, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("muzvibe: failed to parse HTML: %w", err)
	}

	results := make([]domain.ProviderResult, 0, 40)
	rank := 1

	doc.Find("#results .item").Each(func(_ int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find(".title").Text())
		artist := strings.TrimSpace(s.Find(".artist a").Text())
		image := s.Find(".cover img").AttrOr("src", "")
		link := s.Find("a.link").AttrOr("href", "")

		if title == "" || artist == "" {
			return
		}

		if image != "" && image[:2] == "//" {
			image = "https:" + image
		}
		if link != "" && link[:2] == "//" {
			link = "https:" + link
		}

		song := domain.NewSong(title, artist, image, link)
		matchScore := mvp.CalculateBasicMatchScore(*song, query)

		results = append(results, *domain.NewProviderResult(
			*song,
			mvp.Name(),
			matchScore,
			rank,
		))
		rank++
	})

	return results, nil
}
