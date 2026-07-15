package providers

import (
	"net/http"
)

type BaseProvider struct {
	name     string
	priority int
	client   *http.Client
}

func NewBaseProvider(name string, priority int, client *http.Client) *BaseProvider {
	return &BaseProvider{
		name:     name,
		priority: priority,
		client:   client,
	}
}

func (bp *BaseProvider) Name() string {
	return bp.name
}

func (bp *BaseProvider) Priority() int {
	return bp.priority
}

func (bp *BaseProvider) GetClient() *http.Client {
	return bp.client
}

func (bp *BaseProvider) AddBrowserHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}
