package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type MusicFinderService struct {
	sourceLink string
	httpClient *http.Client
}

var _ ports.IMusicFinderService = (*MusicFinderService)(nil)

func NewMusicFinderService() *MusicFinderService {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:     tlsConfig,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &MusicFinderService{
		sourceLink: "https://muzvibe.org/search/",
		httpClient: httpClient,
	}
}

func (mfs *MusicFinderService) FindMusic(ctx context.Context, query string) ([]domain.Song, error) {
	apiURL := mfs.sourceLink + url.QueryEscape(query)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		log.Printf("[MusicFinder] ERROR: Failed to create request for query %q: %v", query, err)
		return nil, fmt.Errorf("search: request creation failed: %w", err)
	}

	resp, err := mfs.httpClient.Do(req)
	if err != nil {
		log.Printf("[MusicFinder] ERROR: HTTP request failed for query %q: %v", query, err)
		return nil, fmt.Errorf("search: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[MusicFinder] ERROR: Unexpected status code %d for query %q", resp.StatusCode, query)
		return nil, fmt.Errorf("search: unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[MusicFinder] ERROR: Failed to parse HTML for query %q: %v", query, err)
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
