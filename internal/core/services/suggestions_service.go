package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type SuggestionsService struct {
	sourceLink string
	client     *http.Client
}

var _ ports.ISuggestionsService = (*SuggestionsService)(nil)

var (
	httpClientOnce sync.Once
	httpClient     *http.Client
)

func getHttpClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClient = &http.Client{}
	})
	return httpClient
}

func NewSuggestionsService() *SuggestionsService {
	return &SuggestionsService{
		client: getHttpClient(),
	}
}

func (ss *SuggestionsService) GetSuggestions(ctx context.Context, query string) ([]string, error) {
	params := url.Values{}
	params.Add("client", "youtube")
	params.Add("ds", "yt")
	params.Add("gl", "RO")
	params.Add("hl", "ro")
	params.Add("q", query)

	apiURL := "https://suggestqueries.google.com/complete/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("suggestions: failed to create request: %w", err)
	}

	resp, err := ss.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("suggestions: failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("suggestions: unexpected status code: %d", resp.StatusCode)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("suggestions: failed to read body: %w", err)
	}

	body := string(payload)
	start := strings.IndexByte(body, '[')
	end := strings.LastIndexByte(body, ']')

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("suggestions: invalid format in response: %s", body)
	}

	var data []any
	if err := json.Unmarshal(payload[start:end+1], &data); err != nil {
		return nil, fmt.Errorf("suggestions: unmarshal error: %w", err)
	}

	if len(data) < 2 {
		return nil, errors.New("suggestions: empty data in response")
	}

	items, ok := data[1].([]any)
	if !ok {
		return nil, errors.New("suggestions: failed to cast suggestions list")
	}

	results := make([]string, 0, len(items))
	for _, rawItem := range items {
		if suggestion, ok := rawItem.([]any); ok && len(suggestion) > 0 {
			if text, ok := suggestion[0].(string); ok {
				results = append(results, text)
			}
		}
	}

	return results, nil
}
