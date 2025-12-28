package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
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
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
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
	req, err := mfs.createSearchRequest(ctx, query)
	if err != nil {
		return nil, err
	}

	resp, err := mfs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: unexpected status code: %d", resp.StatusCode)
	}

	return mfs.parseResults(resp.Body)
}

func (mfs *MusicFinderService) createSearchRequest(ctx context.Context, query string) (*http.Request, error) {
	apiURL := mfs.sourceLink + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("search: request creation failed: %w", err)
	}

	mfs.addBrowserHeaders(req)
	return req, nil
}

func (mfs *MusicFinderService) addBrowserHeaders(req *http.Request) {
	headers := map[string]string{
		"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "en-US,en;q=0.9",
		"Referer":                   "https://muzvibe.org/",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Ch-Ua":                 `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`,
		"Sec-Ch-Ua-Mobile":          "?0",
		"Sec-Ch-Ua-Platform":        `"macOS"`,
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "same-origin",
		"Sec-Fetch-User":            "?1",
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

func (mfs *MusicFinderService) parseResults(body io.Reader) ([]domain.Song, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("search: failed to parse HTML: %w", err)
	}

	var results []domain.Song
	doc.Find("#results .item").Each(func(_ int, s *goquery.Selection) {
		results = append(results, *domain.NewSong(
			s.Find(".title").Text(),
			s.Find(".artist").Text(),
			s.Find("img").AttrOr("src", ""),
			s.Find(".link").AttrOr("href", ""),
		))
	})

	return results, nil
}
