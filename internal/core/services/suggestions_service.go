package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

type SuggestionsService struct {
	client *http.Client
}

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
	apiURL := "https://suggestqueries.google.com/complete/search?client=youtube&ds=yt&gl=RO&hl=ro&q=" + url.QueryEscape(query)

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

	start := bytes.IndexByte(payload, '[')
	end := bytes.LastIndexByte(payload, ']')

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("suggestions: invalid format in response")
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
