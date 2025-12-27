package services

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type MusicFinderService struct {
	sourceLink string
}

var _ ports.IMusicFinderService = (*MusicFinderService)(nil)

func NewMusicFinderService() *MusicFinderService {
	return &MusicFinderService{
		sourceLink: "https://muzvibe.org/search/",
	}
}

func (mfs *MusicFinderService) FindMusic(ctx context.Context, query string) ([]domain.Song, error) {
	apiURL := mfs.sourceLink + url.QueryEscape(query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("search: request creation failed: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("search: failed to parse HTML: %w", err)
	}

	results := make([]domain.Song, 0, 40)
	doc.Find("#results .item").Each(func(i int, s *goquery.Selection) {
		song := domain.NewSong(
			s.Find(".title").Text(),
			s.Find(".artist").Text(),
			s.Find("img").AttrOr("src", ""),
			s.Find(".link").AttrOr("href", ""),
		)
		results = append(results, *song)
	})

	return results, nil
}
