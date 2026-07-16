package handlers

import (
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
}

type sourceStatus struct {
	Name string `json:"name"`
	Host string `json:"host"`
	OK   bool   `json:"ok"`
	Ms   int64  `json:"ms"`
}

var musicSources = []sourceSpec{
	{Name: "Mp3mn", Host: "mp3mn.net", URL: "https://mp3mn.net/"},
}

type HealthHandler struct {
	client *http.Client
}

func NewHealthHandler(client *http.Client) *HealthHandler {
	return &HealthHandler{client: client}
}

func (hh *HealthHandler) GetHealth(c fiber.Ctx) error {
	return c.JSON("Pong")
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
	req.Header.Set("User-Agent", "FindVibeHealth/1.0")

	resp, err := hh.client.Do(req)
	st.Ms = time.Since(start).Milliseconds()
	if err != nil {
		return st
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
	st.OK = resp.StatusCode > 0 && resp.StatusCode < 500
	return st
}
