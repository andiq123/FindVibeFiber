package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/andiq123/FindVibeFiber/internals/core/ports"
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
		sourceLink: "https://clients1.google.com/complete/search?client=youtube&gs_ri=youtube&ds=yt&q=",
		client:     getHttpClient(),
	}
}

func (s *SuggestionsService) GetSuggestions(query string) ([]string, error) {
	fullLink := fmt.Sprintf("%s%s", s.sourceLink, query)

	resp, err := s.client.Get(fullLink)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	startIndex := strings.IndexByte(string(bodyBytes), '[')
	endIndex := strings.LastIndexByte(string(bodyBytes), ']')

	if startIndex == -1 || endIndex == -1 {
		return nil, errors.New("failed to find suggestions")
	}

	var data []any
	if err := json.Unmarshal(bodyBytes[startIndex:endIndex+1], &data); err != nil {
		return nil, err
	}

	if len(data) < 2 {
		return nil, errors.New("invalid suggestions format")
	}

	items, ok := data[1].([]any)
	if !ok {
		return nil, errors.New("failed to parse suggestions")
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
