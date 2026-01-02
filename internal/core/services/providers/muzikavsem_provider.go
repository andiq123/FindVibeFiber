package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type musMeta struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Img    string `json:"img"`
	ID     string `json:"id"`
}

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

	results := make([]domain.ProviderResult, 0, 40)
	rank := 1

	doc.Find("ul.top-tracks__list li.top-tracks__item").Each(func(_ int, s *goquery.Selection) {
		metaJSON := s.AttrOr("data-musmeta", "")
		var meta musMeta
		if metaJSON != "" {
			_ = json.Unmarshal([]byte(metaJSON), &meta)
		}

		title := s.Find(".top-tracks__track").Text()
		if title == "" {
			title = meta.Title
		}
		artist := s.Find(".top-tracks__artist").Text()
		if artist == "" {
			artist = meta.Artist
		}

		downloadPath := s.Find("a.top-tracks__download-btn").AttrOr("href", "")
		if downloadPath == "" {
			downloadPath = meta.URL
		}

		if downloadPath == "" {
			return
		}

		image := meta.Img
		if image == "" {
			imageStyle := s.Find(".top-tracks__img").AttrOr("style", "")
			image = mvp.extractImageFromStyle(imageStyle)
		}

		if image != "" && image[0] == '/' {
			image = "https://new.muzikavsem.org" + image
		}

		if !strings.HasPrefix(downloadPath, "/") {
			downloadPath = "/" + downloadPath
		}

		link := "https://new.muzikavsem.org" + downloadPath

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

func (mvp *MuzikaVsemProvider) extractImageFromStyle(styleAttr string) string {
	if styleAttr == "" {
		return ""
	}

	start := strings.Index(styleAttr, "url(")
	if start == -1 {
		return ""
	}
	start += 4

	end := strings.Index(styleAttr[start:], ")")
	if end == -1 {
		return ""
	}

	imageURL := styleAttr[start : start+end]
	imageURL = trimQuotes(imageURL)

	return imageURL
}

func trimQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}
