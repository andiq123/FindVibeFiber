package handlers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

type sourceSpec struct {
	Name string `json:"name"`
	Host string `json:"host"`
	URL  string `json:"-"`
	// requireMarkup — when set, body must contain one of these markers (search scrape path).
	Markers [][]byte `json:"-"`
}

type sourceStatus struct {
	Name string `json:"name"`
	Host string `json:"host"`
	OK   bool   `json:"ok"`
	Ms   int64  `json:"ms"`
}

var musicSources = []sourceSpec{
	{
		Name: "MuzJam",
		Host: "muzjam.org",
		// Probe the real scrape path — homepage <500 was a false positive.
		URL:     "https://muzjam.org/search/test",
		Markers: [][]byte{[]byte(`id="results"`), []byte(`class="item"`), []byte(`a.link`)},
	},
	{
		Name:    "Mp3mn",
		Host:    "mp3mn.net",
		URL:     "https://mp3mn.net/",
		Markers: nil, // homepage reachability only
	},
}

type HealthHandler struct {
	client *http.Client
}

func NewHealthHandler(client *http.Client) *HealthHandler {
	return &HealthHandler{client: client}
}

func (hh *HealthHandler) GetSources(c fiber.Ctx) error {
	out := make([]sourceStatus, len(musicSources))
	var wg sync.WaitGroup
	for i, s := range musicSources {
		wg.Add(1)
		go func(i int, s sourceSpec) {
			defer wg.Done()
			out[i] = hh.probe(s)
		}(i, s)
	}
	wg.Wait()
	return c.JSON(fiber.Map{"sources": out})
}

func (hh *HealthHandler) probe(s sourceSpec) sourceStatus {
	st := sourceStatus{Name: s.Name, Host: s.Host}
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		st.Ms = time.Since(start).Milliseconds()
		return st
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := hh.client.Do(req)
	st.Ms = time.Since(start).Milliseconds()
	if err != nil {
		return st
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		return st
	}
	if len(s.Markers) == 0 {
		st.OK = true
		return st
	}
	for _, m := range s.Markers {
		if bytes.Contains(body, m) {
			st.OK = true
			return st
		}
	}
	return st
}
