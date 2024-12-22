package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/andiq123/FindVibeFiber/internals/core/ports"
)

type SuggestionsService struct {
	sourceLink string
}

var _ ports.ISuggestionsService = (*SuggestionsService)(nil)

func NewSuggestionsService() *SuggestionsService {
	return &SuggestionsService{
		sourceLink: "https://clients1.google.com/complete/search?client=youtube&gs_ri=youtube&ds=yt&q=",
	}
}

func (s *SuggestionsService) GetSuggestions(query string) ([]string, error) {
	resp, err := http.Get(s.sourceLink + query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch suggestions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	var data []any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(data) < 2 {
		return nil, errors.New("invalid response structure")
	}

	items, ok := data[1].([]any)
	if !ok {
		return nil, errors.New("invalid suggestions data format")
	}

	results := make([]string, 0, len(items))
	for _, item := range items {
		if suggestion, ok := item.([]any); ok && len(suggestion) > 0 {
			if str, ok := suggestion[0].(string); ok {
				results = append(results, str)
			}
		}
	}

	return results, nil
}
