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
	"strings"
	"unicode/utf8"
)

// Google's latin-1 suggest feed uses ã/Ã where Romanian wants ă/Ă.
var roSuggestFix = strings.NewReplacer("ã", "ă", "Ã", "Ă")

type SuggestionsService struct {
	client *http.Client
}

func NewSuggestionsService(client *http.Client) *SuggestionsService {
	return &SuggestionsService{client: client}
}

func (ss *SuggestionsService) GetSuggestions(ctx context.Context, query string) ([]string, error) {
	// firefox client: plain JSON array (no JSONP). Still latin-1 for RO diacritics.
	apiURL := "https://suggestqueries.google.com/complete/search?client=firefox&hl=ro&gl=RO&q=" + url.QueryEscape(query)

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

	// Google emits ISO-8859-1; encoding/json expects UTF-8.
	raw := latin1ToUTF8(payload[start : end+1])

	var data []any
	if err := json.Unmarshal(raw, &data); err != nil {
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
		switch v := rawItem.(type) {
		case string:
			if v != "" {
				results = append(results, roSuggestFix.Replace(v))
			}
		case []any:
			if len(v) > 0 {
				if text, ok := v[0].(string); ok && text != "" {
					results = append(results, roSuggestFix.Replace(text))
				}
			}
		}
	}
	return results, nil
}

// latin1ToUTF8 maps each byte to a Unicode code point (ISO-8859-1).
func latin1ToUTF8(b []byte) []byte {
	out := make([]byte, 0, len(b)+len(b)/4)
	for _, c := range b {
		out = utf8.AppendRune(out, rune(c))
	}
	return out
}
