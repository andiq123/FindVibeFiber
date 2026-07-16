package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/utils"
)

const (
	coverCacheTTL = 24 * time.Hour
	coverFillConc = 8
	coverCacheMax = 10_000 // ponytail: wipe when huge; LRU if hit rate matters
)

type coverEntry struct {
	url string
	exp time.Time
}

// CoverService: iTunes artwork lookup with process-wide cache (hits + misses).
type CoverService struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]coverEntry
}

func NewCoverService(client *http.Client) *CoverService {
	return &CoverService{client: client, cache: make(map[string]coverEntry)}
}

// Lookup returns artwork URL for a free-text query, or "".
func (cs *CoverService) Lookup(ctx context.Context, q string) string {
	if cs == nil {
		return ""
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	key := utils.NormalizeString(q)
	if img, ok := cs.get(key); ok {
		return img
	}
	img := fetchItunesCover(ctx, cs.client, q)
	cs.put(key, img)
	return img
}

// FillSongs sets Image on songs that lack one (parallel, cached).
func (cs *CoverService) FillSongs(ctx context.Context, songs []domain.Song) {
	if cs == nil || len(songs) == 0 {
		return
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, coverFillConc)
	for i := range songs {
		if strings.TrimSpace(songs[i].Image) != "" {
			continue
		}
		q := strings.TrimSpace(songs[i].Artist + " " + songs[i].Title)
		if q == "" {
			continue
		}
		wg.Add(1)
		go func(i int, q string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if img := cs.Lookup(ctx, q); img != "" {
				songs[i].Image = img
			}
		}(i, q)
	}
	wg.Wait()
}

func (cs *CoverService) get(key string) (string, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	e, ok := cs.cache[key]
	if !ok {
		return "", false
	}
	if time.Now().After(e.exp) {
		delete(cs.cache, key)
		return "", false
	}
	return e.url, true
}

func (cs *CoverService) put(key, img string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if len(cs.cache) >= coverCacheMax {
		cs.cache = make(map[string]coverEntry)
	}
	cs.cache[key] = coverEntry{url: img, exp: time.Now().Add(coverCacheTTL)}
}

func fetchItunesCover(ctx context.Context, client *http.Client, q string) string {
	if client == nil {
		return ""
	}
	apiURL := "https://itunes.apple.com/search?media=music&entity=song&limit=1&term=" + url.QueryEscape(q)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var data struct {
		Results []struct {
			ArtworkURL100 string `json:"artworkUrl100"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	if len(data.Results) == 0 || data.Results[0].ArtworkURL100 == "" {
		return ""
	}
	return strings.Replace(data.Results[0].ArtworkURL100, "100x100", "600x600", 1)
}
