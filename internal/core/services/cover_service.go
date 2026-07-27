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
	"golang.org/x/sync/singleflight"
)

const (
	coverCacheTTL = 24 * time.Hour
	// ponytail: empty iTunes misses must not lock out retries for a full day
	coverMissTTL  = 15 * time.Minute
	coverFillConc = 8
	coverCacheMax = 10_000
	coverFetchCap = 8 * time.Second
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
	sf     singleflight.Group
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

	v, _, _ := cs.sf.Do(key, func() (any, error) {
		if img, ok := cs.get(key); ok {
			return img, nil
		}
		// Shared fill must not die with the first client's cancel.
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), coverFetchCap)
		defer cancel()
		img := fetchItunesCover(fctx, cs.client, q)
		cs.put(key, img)
		return img, nil
	})
	img, _ := v.(string)
	return img
}

// FillSongs sets Image on songs that lack one (parallel, cached). Honors ctx deadline.
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
		if err := ctx.Err(); err != nil {
			return
		}
		wg.Add(1)
		go func(i int, q string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
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
	ttl := coverCacheTTL
	if img == "" {
		ttl = coverMissTTL
	}
	cs.cache[key] = coverEntry{url: img, exp: time.Now().Add(ttl)}
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
