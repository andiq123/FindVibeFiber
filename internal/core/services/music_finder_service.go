package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
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

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://muzvibe.org/")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")

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

	// Debug: Read and log the response body to see what we're actually getting
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[MusicFinder] ERROR: Failed to read response body for query %q: %v", query, err)
		return nil, fmt.Errorf("search: failed to read response body: %w", err)
	}
	// Restore the body for goquery
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	log.Printf("[MusicFinder] DEBUG: HTML Content for query %q:\n%s", query, string(bodyBytes))

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[MusicFinder] ERROR: Failed to parse HTML for query %q: %v", query, err)
		return nil, fmt.Errorf("search: failed to parse HTML: %w", err)
	}

	// Debug: Log what we're looking for and what we find
	resultsContainer := doc.Find("#results")
	log.Printf("[MusicFinder] DEBUG: Found #results container: %v, items count: %d", resultsContainer.Length() > 0, doc.Find("#results .item").Length())

	// Debug: Log first item details if any exist
	firstItem := doc.Find("#results .item").First()
	if firstItem.Length() > 0 {
		title := firstItem.Find(".title").Text()
		artist := firstItem.Find(".artist").Text()
		log.Printf("[MusicFinder] DEBUG: First item - title: %q, artist: %q", title, artist)
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

	log.Printf("[MusicFinder] DEBUG: Total songs found: %d for query %q", len(results), query)

	return results, nil
}
