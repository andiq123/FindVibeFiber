package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/constants"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"golang.org/x/sync/singleflight"
)

const (
	searchCacheTTL = 2 * time.Minute
	searchCacheCap = 256
	// Oversample Last.fm so provider mapping attrition still fills MaxResults.
	searchCatalogOversample = 2
	searchMapPeek           = 8
)

type searchCacheEntry struct {
	resp *domain.SearchResponse
	at   time.Time
}

// catalogSearcher is Last.fm discovery (tests inject a stub).
type catalogSearcher interface {
	Configured() bool
	Search(ctx context.Context, query string, page, limit int) (CatalogPage, error)
	TopAlbums(ctx context.Context, artist string, limit int) ([]domain.ArtistAlbum, error)
}

type SearchService struct {
	providers     []ports.IMusicProvider
	config        *domain.SearchConfig
	searchTimeout time.Duration
	catalog       catalogSearcher

	cacheMu sync.Mutex
	cache   map[string]searchCacheEntry
	sf      singleflight.Group
}

func NewSearchService(
	providers []ports.IMusicProvider,
	config *domain.SearchConfig,
	timeout time.Duration,
	catalog catalogSearcher,
) *SearchService {
	if config == nil {
		config = domain.DefaultSearchConfig()
	}
	if timeout <= 0 {
		timeout = time.Duration(constants.DefaultSearchTimeout) * time.Second
	}

	return &SearchService{
		providers:     providers,
		config:        config,
		searchTimeout: timeout,
		catalog:       catalog,
		cache:         make(map[string]searchCacheEntry),
	}
}

// Search: Last.fm discovers songs/artists → providers map playable URLs → only successful maps.
func (ss *SearchService) Search(ctx context.Context, query string, page int) (*domain.SearchResponse, error) {
	if ss.catalog == nil || !ss.catalog.Configured() {
		return nil, fmt.Errorf("search: %w", domain.ErrUnavailable)
	}
	if len(ss.providers) == 0 {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}

	text := strings.TrimSpace(query)
	if text == "" {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}
	if page < 1 {
		page = 1
	}

	key := searchCacheKey(text, page)
	if hit := ss.cacheGet(key); hit != nil {
		return cloneSearchResponse(hit), nil
	}

	v, err, _ := ss.sf.Do(key, func() (any, error) {
		if hit := ss.cacheGet(key); hit != nil {
			return hit, nil
		}
		resp, err := ss.searchUncached(ctx, text, page)
		if err != nil {
			return nil, err
		}
		if resp != nil && len(resp.Songs) > 0 {
			ss.cachePut(key, resp)
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	resp, _ := v.(*domain.SearchResponse)
	if resp == nil {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}
	return cloneSearchResponse(resp), nil
}

func (ss *SearchService) searchUncached(ctx context.Context, text string, page int) (*domain.SearchResponse, error) {
	maxResults := ss.config.MaxResults
	if maxResults <= 0 {
		maxResults = constants.DefaultMaxSearchResults
	}
	limit := maxResults * searchCatalogOversample
	if limit < maxResults {
		limit = maxResults
	}

	pageData, err := ss.catalog.Search(ctx, text, page, limit)
	if err != nil {
		return nil, err
	}
	songs := ss.mapCatalogHits(ctx, pageData.Hits, maxResults)
	resp := domain.NewSearchResponse(songs, pageData.Pagination)

	// Discovery chrome only on page 1 — artists + top artist's albums.
	if page == 1 && len(pageData.Artists) > 0 {
		resp.Artists = make([]domain.SearchArtist, 0, len(pageData.Artists))
		for _, a := range pageData.Artists {
			resp.Artists = append(resp.Artists, domain.SearchArtist{Name: a.Name, Image: a.Image})
		}
		top := pageData.Artists[0].Name
		if albums, err := ss.catalog.TopAlbums(ctx, top, catalogAlbumsForTopArtist); err == nil && len(albums) > 0 {
			resp.Albums = albums
		}
	}
	return resp, nil
}

// mapCatalogHits resolves Last.fm rows through providers in Last.fm order; drops misses.
func (ss *SearchService) mapCatalogHits(ctx context.Context, hits []CatalogHit, capN int) []domain.Song {
	if capN < 1 || len(hits) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type slot struct {
		i    int
		song domain.Song
		ok   bool
	}
	ch := make(chan slot, len(hits))
	var wg sync.WaitGroup
	sem := make(chan struct{}, constants.DefaultResolveConcurrency)

	for i, hit := range hits {
		wg.Add(1)
		go func(i int, hit CatalogHit) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				ch <- slot{i: i}
				return
			}
			song, ok := ss.mapOne(ctx, hit)
			ch <- slot{i: i, song: song, ok: ok}
		}(i, hit)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	byIdx := make(map[int]domain.Song, len(hits))
	decided := make([]bool, len(hits))
	for s := range ch {
		decided[s.i] = true
		if s.ok {
			byIdx[s.i] = s.song
		}
		prefix := 0
		for prefix < len(hits) && decided[prefix] {
			prefix++
		}
		if out := pickMappedPrefix(byIdx, prefix, capN); len(out) >= capN {
			cancel()
			return out
		}
	}
	return pickMappedPrefix(byIdx, len(hits), capN)
}

func pickMappedPrefix(byIdx map[int]domain.Song, n, capN int) []domain.Song {
	seen := map[string]struct{}{}
	out := make([]domain.Song, 0, capN)
	for i := 0; i < n && len(out) < capN; i++ {
		song, ok := byIdx[i]
		if !ok {
			continue
		}
		k := SongKey(song.Artist, song.Title)
		if k == "" {
			continue
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, song)
	}
	return out
}

func (ss *SearchService) mapOne(ctx context.Context, hit CatalogHit) (domain.Song, bool) {
	songs, err := ss.SearchFirst(ctx, hit.Artist+" "+hit.Title, searchMapPeek)
	if err != nil || len(songs) == 0 {
		return domain.Song{}, false
	}
	song, ok := PickPlayableSong(hit.Artist, hit.Title, songs, "", searchMapPeek)
	if !ok {
		return domain.Song{}, false
	}
	if strings.TrimSpace(song.Image) == "" && strings.TrimSpace(hit.Image) != "" {
		song.Image = hit.Image
	}
	return song, true
}

// SearchFirst fans out by priority and returns as soon as the best available
// playable hit cannot be beaten by any still-in-flight higher-priority provider.
func (ss *SearchService) SearchFirst(ctx context.Context, query string, limit int) ([]domain.Song, error) {
	if len(ss.providers) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = searchMapPeek
	}

	text := strings.TrimSpace(query)
	if text == "" {
		return nil, nil
	}

	providers := append([]ports.IMusicProvider(nil), ss.providers...)
	sort.SliceStable(providers, func(i, j int) bool {
		return providers[i].Priority() > providers[j].Priority()
	})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type outcome struct {
		idx   int
		songs []domain.Song
	}
	ch := make(chan outcome, len(providers))
	var wg sync.WaitGroup
	for i, p := range providers {
		wg.Add(1)
		go func(i int, p ports.IMusicProvider) {
			defer wg.Done()
			pctx, pcancel := context.WithTimeout(ctx, ss.searchTimeout)
			defer pcancel()
			got, err := p.SearchWithPage(pctx, text, 1)
			if err != nil {
				if !isBenignSearchErr(err) {
					utils.GetLogger().Warn("provider search-first failed", "provider", p.Name(), "query", text, "error", err)
				}
				ch <- outcome{idx: i}
				return
			}
			ch <- outcome{idx: i, songs: playableSongs(got, limit)}
		}(i, p)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	unfinished := make([]bool, len(providers))
	for i := range unfinished {
		unfinished[i] = true
	}

	var best []domain.Song
	bestPri := -1
	for o := range ch {
		unfinished[o.idx] = false
		if len(o.songs) > 0 {
			pri := providers[o.idx].Priority()
			if pri > bestPri {
				bestPri = pri
				best = o.songs
			}
		}
		if len(best) == 0 {
			continue
		}
		higherPending := false
		for j, p := range providers {
			if unfinished[j] && p.Priority() > bestPri {
				higherPending = true
				break
			}
		}
		if !higherPending {
			cancel()
			for range ch {
			}
			return best, nil
		}
	}
	return best, nil
}

func playableSongs(results []domain.ProviderResult, limit int) []domain.Song {
	songs := make([]domain.Song, 0, limit)
	for i := range results {
		if len(songs) >= limit {
			break
		}
		s := results[i].Song
		s.Link = utils.UpgradeHTTPS(s.Link)
		s.Image = utils.UpgradeHTTPS(s.Image)
		if !strings.HasPrefix(s.Link, "https://") {
			continue
		}
		songs = append(songs, s)
	}
	return songs
}

func searchCacheKey(text string, page int) string {
	return utils.NormalizeString(text) + "|" + strconv.Itoa(page)
}

func (ss *SearchService) cacheGet(key string) *domain.SearchResponse {
	ss.cacheMu.Lock()
	defer ss.cacheMu.Unlock()
	e, ok := ss.cache[key]
	if !ok || time.Since(e.at) > searchCacheTTL {
		return nil
	}
	return e.resp
}

func (ss *SearchService) cachePut(key string, resp *domain.SearchResponse) {
	if resp == nil || len(resp.Songs) == 0 {
		return
	}
	ss.cacheMu.Lock()
	defer ss.cacheMu.Unlock()
	if len(ss.cache) >= searchCacheCap {
		ss.cache = make(map[string]searchCacheEntry, searchCacheCap/2)
	}
	ss.cache[key] = searchCacheEntry{resp: cloneSearchResponse(resp), at: time.Now()}
}

func cloneSearchResponse(r *domain.SearchResponse) *domain.SearchResponse {
	if r == nil {
		return nil
	}
	songs := append([]domain.Song(nil), r.Songs...)
	artists := append([]domain.SearchArtist(nil), r.Artists...)
	albums := append([]domain.ArtistAlbum(nil), r.Albums...)
	var pag *domain.PaginationInfo
	if r.Pagination != nil {
		p := *r.Pagination
		pag = &p
	}
	return &domain.SearchResponse{
		Songs:      songs,
		Artists:    artists,
		Albums:     albums,
		Pagination: pag,
	}
}

func isBenignSearchErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
