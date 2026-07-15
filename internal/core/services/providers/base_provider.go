package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const maxHTMLBytes = 1 << 20

type BaseProvider struct {
	name     string
	priority int
	client   *http.Client
}

func NewBaseProvider(name string, priority int, client *http.Client) *BaseProvider {
	return &BaseProvider{name: name, priority: priority, client: client}
}

func (bp *BaseProvider) Name() string  { return bp.name }
func (bp *BaseProvider) Priority() int { return bp.priority }

func (bp *BaseProvider) fetchDocument(ctx context.Context, rawURL, referer string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: request: %w", bp.name, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := bp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: fetch: %w", bp.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: status %d", bp.name, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(io.LimitReader(resp.Body, maxHTMLBytes))
	if err != nil {
		return nil, fmt.Errorf("%s: parse: %w", bp.name, err)
	}
	return doc, nil
}

func absoluteURL(raw string) string {
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return raw
}

func text(s *goquery.Selection) string {
	return strings.TrimSpace(s.Text())
}
