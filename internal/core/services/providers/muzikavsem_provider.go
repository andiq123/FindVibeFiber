package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type MuzikaVsemProvider struct {
	*BaseProvider
	sourceURL string
}

var _ ports.IMusicProvider = (*MuzikaVsemProvider)(nil)

func NewMuzikaVsemProvider(client *http.Client) *MuzikaVsemProvider {
	return &MuzikaVsemProvider{
		BaseProvider: NewBaseProvider("MuzikaVsem", 7, client),
		sourceURL:    "https://new.muzikavsem.org/search/",
	}
}

func (mvp *MuzikaVsemProvider) Search(ctx context.Context, query string) ([]domain.ProviderResult, error) {
	req, err := mvp.createSearchRequest(ctx, query)
	if err != nil {
		return nil, err
	}

	resp, err := mvp.GetClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("muzikavsem: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("muzikavsem: unexpected status code: %d", resp.StatusCode)
	}

	return mvp.parseResults(resp.Body, query)
}

func (mvp *MuzikaVsemProvider) createSearchRequest(ctx context.Context, query string) (*http.Request, error) {
	apiURL := mvp.sourceURL + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("muzikavsem: request creation failed: %w", err)
	}

	mvp.AddBrowserHeaders(req, "https://new.muzikavsem.org/")
	return req, nil
}

func (mvp *MuzikaVsemProvider) parseResults(body io.Reader, query string) ([]domain.ProviderResult, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("muzikavsem: failed to parse HTML: %w", err)
	}

	var results []domain.ProviderResult
	rank := 1

	doc.Find("ul.top-tracks__list li.top-tracks__item").Each(func(_ int, s *goquery.Selection) {
		title := s.Find(".top-tracks__track").Text()
		artist := s.Find(".top-tracks__artist").Text()

		imageStyle := s.Find(".top-tracks__img").AttrOr("style", "")
		image := mvp.extractImageURL(imageStyle)

		link := s.Find("a.top-tracks__caption").AttrOr("href", "")
		if link != "" && link[0] == '/' {
			link = "https://new.muzikavsem.org" + link
		}

		if title != "" && artist != "" {
			song := domain.NewSong(title, artist, image, link)
			matchScore := mvp.CalculateBasicMatchScore(*song, query)

			results = append(results, *domain.NewProviderResult(
				*song,
				mvp.Name(),
				matchScore,
				rank,
			))
			rank++
		}
	})

	return results, nil
}

func (mvp *MuzikaVsemProvider) extractImageURL(styleAttr string) string {
	if styleAttr == "" {
		return ""
	}

	start := -1
	for i := 0; i < len(styleAttr)-3; i++ {
		if styleAttr[i:i+4] == "url(" {
			start = i + 4
			break
		}
	}

	if start == -1 {
		return ""
	}

	end := -1
	for i := start; i < len(styleAttr); i++ {
		if styleAttr[i] == ')' {
			end = i
			break
		}
	}

	if end == -1 {
		return ""
	}

	imageURL := styleAttr[start:end]
	imageURL = trimQuotes(imageURL)

	if imageURL != "" && imageURL[0] == '/' {
		imageURL = "https://new.muzikavsem.org" + imageURL
	}

	return imageURL
}

func trimQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}
