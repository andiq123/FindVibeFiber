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
	// Soft wait: after this, finish early once we have enough playable rows.
	searchCollectSoftMin = 800 * time.Millisecond
)

type searchCacheEntry struct {
	resp *domain.SearchResponse
	at   time.Time
}

type SearchService struct {
	providers     []ports.IMusicProvider
	priorities    map[string]int
	ranker        *SearchRanker
	config        *domain.SearchConfig
	searchTimeout time.Duration

	cacheMu sync.Mutex
	cache   map[string]searchCacheEntry
	sf      singleflight.Group
}

func NewSearchService(providers []ports.IMusicProvider, config *domain.SearchConfig, timeout time.Duration) *SearchService {
	if config == nil {
		config = domain.DefaultSearchConfig()
	}
	if timeout <= 0 {
		timeout = time.Duration(constants.DefaultSearchTimeout) * time.Second
	}

	priorities := make(map[string]int, len(providers))
	for _, p := range providers {
		priorities[p.Name()] = p.Priority()
	}

	return &SearchService{
		providers:     providers,
		priorities:    priorities,
		ranker:        NewSearchRanker(config.RankingWeights),
		config:        config,
		searchTimeout: timeout,
		cache:         make(map[string]searchCacheEntry),
	}
}

func (ss *SearchService) Search(ctx context.Context, query string, page int) (*domain.SearchResponse, error) {
	if len(ss.providers) == 0 {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}

	parsed := ParseSearchQuery(query)
	if parsed.Text == "" {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}

	providers := ss.providers
	if parsed.HasYearFilter() {
		providers = yearFilterProviders(ss.providers)
		if len(providers) == 0 {
			return domain.NewSearchResponse([]domain.Song{}, nil), nil
		}
	}

	key := searchCacheKey(parsed, page)
	if hit := ss.cacheGet(key); hit != nil {
		return cloneSearchResponse(hit), nil
	}

	v, err, _ := ss.sf.Do(key, func() (any, error) {
		if hit := ss.cacheGet(key); hit != nil {
			return hit, nil
		}
		resp, err := ss.searchUncached(ctx, parsed, page, providers)
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

func (ss *SearchService) searchUncached(
	ctx context.Context,
	parsed ParsedSearchQuery,
	page int,
	providers []ports.IMusicProvider,
) (*domain.SearchResponse, error) {
	results := ss.collect(ctx, parsed, page, providers)
	if len(results) == 0 {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}

	results = dedupe(results)
	scored := ss.ranker.RankResults(results, parsed.Text, ss.priorities)

	n := ss.config.MaxResults
	if n <= 0 {
		n = len(scored)
	}

	songs := make([]domain.Song, 0, n)
	for i := 0; i < len(scored) && len(songs) < n; i++ {
		s := scored[i].Result.Song
		s.Link = upgradeHTTPS(s.Link)
		s.Image = upgradeHTTPS(s.Image)
		if !strings.HasPrefix(s.Link, "https://") {
			continue
		}
		songs = append(songs, s)
	}
	return domain.NewSearchResponse(songs, pickPagination(scored)), nil
}

// SearchFirst fans out by priority and returns as soon as the best available
// playable hit cannot be beaten by any still-in-flight higher-priority provider.
func (ss *SearchService) SearchFirst(ctx context.Context, query string, limit int) ([]domain.Song, error) {
	if len(ss.providers) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = recommendSearchPeekDefault
	}

	text := ParseSearchQuery(query).Text
	if text == "" {
		text = strings.TrimSpace(query)
	}
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

const recommendSearchPeekDefault = 8

func playableSongs(results []domain.ProviderResult, limit int) []domain.Song {
	songs := make([]domain.Song, 0, limit)
	for i := range results {
		if len(songs) >= limit {
			break
		}
		s := results[i].Song
		s.Link = upgradeHTTPS(s.Link)
		s.Image = upgradeHTTPS(s.Image)
		if !strings.HasPrefix(s.Link, "https://") {
			continue
		}
		songs = append(songs, s)
	}
	return songs
}

func upgradeHTTPS(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "http://") {
		return "https://" + strings.TrimPrefix(raw, "http://")
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	return raw
}

// collect fans out in parallel. Hard deadline = searchTimeout.
// Soft deadline (half timeout, min 800ms): cancel stragglers once we have enough playable rows.
// Enough for early cancel at soft = half MaxResults (min 8); at any time = MaxResults.
func (ss *SearchService) collect(ctx context.Context, q ParsedSearchQuery, page int, providers []ports.IMusicProvider) []domain.ProviderResult {
	n := len(providers)
	if n == 0 {
		return nil
	}

	maxResults := ss.config.MaxResults
	if maxResults <= 0 {
		maxResults = constants.DefaultMaxSearchResults
	}
	earlyMin := maxResults / 2
	if earlyMin < 8 {
		earlyMin = 8
	}
	if earlyMin > maxResults {
		earlyMin = maxResults
	}

	pctx, cancel := context.WithTimeout(ctx, ss.searchTimeout)
	defer cancel()

	ch := make(chan []domain.ProviderResult, n)
	for _, p := range providers {
		go func(p ports.IMusicProvider) {
			got, err := searchProvider(pctx, p, q, page)
			if err != nil {
				if !isBenignSearchErr(err) {
					utils.GetLogger().Warn("provider search failed", "provider", p.Name(), "query", q.Text, "error", err)
				}
				ch <- nil
				return
			}
			ch <- got
		}(p)
	}

	softWait := ss.searchTimeout / 2
	if softWait < searchCollectSoftMin {
		softWait = searchCollectSoftMin
	}
	if softWait > ss.searchTimeout {
		softWait = ss.searchTimeout
	}
	soft := time.NewTimer(softWait)
	defer soft.Stop()

	out := make([]domain.ProviderResult, 0, 40*n)
	pending := n
	softOpen := true

	drain := func() {
		for pending > 0 {
			got := <-ch
			pending--
			if len(got) > 0 {
				out = append(out, got...)
			}
		}
	}

	take := func(got []domain.ProviderResult) {
		pending--
		if len(got) > 0 {
			out = append(out, got...)
		}
	}

	for pending > 0 {
		if softOpen {
			select {
			case got := <-ch:
				take(got)
				if playableUnique(out) >= maxResults {
					cancel()
					drain()
					return out
				}
			case <-soft.C:
				softOpen = false
				if playableUnique(out) >= earlyMin {
					cancel()
					drain()
					return out
				}
			case <-pctx.Done():
				cancel()
				drain()
				return out
			}
			continue
		}

		select {
		case got := <-ch:
			take(got)
			// Past soft deadline — finish once we have a useful page.
			if playableUnique(out) >= earlyMin {
				cancel()
				drain()
				return out
			}
		case <-pctx.Done():
			cancel()
			drain()
			return out
		}
	}
	return out
}

func playableUnique(results []domain.ProviderResult) int {
	seen := make(map[string]struct{}, len(results))
	for i := range results {
		link := upgradeHTTPS(results[i].Song.Link)
		if !strings.HasPrefix(link, "https://") {
			continue
		}
		key := utils.NormalizeString(results[i].Song.Title) + "|" + utils.NormalizeString(results[i].Song.Artist)
		if key == "|" {
			continue
		}
		seen[key] = struct{}{}
	}
	return len(seen)
}

func searchCacheKey(q ParsedSearchQuery, page int) string {
	key := utils.NormalizeString(q.Text) + "|" + strconv.Itoa(page)
	if q.HasYearFilter() {
		key += fmt.Sprintf("|y%d-%d", q.YearFrom, q.YearTo)
	}
	return key
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
	var pag *domain.PaginationInfo
	if r.Pagination != nil {
		p := *r.Pagination
		pag = &p
	}
	return domain.NewSearchResponse(songs, pag)
}

func searchProvider(ctx context.Context, p ports.IMusicProvider, q ParsedSearchQuery, page int) ([]domain.ProviderResult, error) {
	if q.HasYearFilter() {
		if yf, ok := p.(ports.YearFilterProvider); ok {
			return yf.SearchWithPageYears(ctx, q.Text, page, q.YearFrom, q.YearTo)
		}
		return nil, nil
	}
	return p.SearchWithPage(ctx, q.Text, page)
}

func yearFilterProviders(all []ports.IMusicProvider) []ports.IMusicProvider {
	out := make([]ports.IMusicProvider, 0, len(all))
	for _, p := range all {
		if _, ok := p.(ports.YearFilterProvider); ok {
			out = append(out, p)
		}
	}
	return out
}

func dedupe(results []domain.ProviderResult) []domain.ProviderResult {
	seen := make(map[string]int, len(results))
	out := make([]domain.ProviderResult, 0, len(results))

	for i := range results {
		key := utils.NormalizeString(results[i].Song.Title) + "|" + utils.NormalizeString(results[i].Song.Artist)
		if idx, ok := seen[key]; ok {
			mergeDuplicate(&out[idx], &results[i])
			continue
		}
		seen[key] = len(out)
		out = append(out, results[i])
	}
	return out
}

// mergeDuplicate combines both sources into one row (cover, best link, combined provider).
func mergeDuplicate(dst, src *domain.ProviderResult) {
	if dst.Song.Image == "" && src.Song.Image != "" {
		dst.Song.Image = src.Song.Image
	}
	if src.Song.Link != "" && (dst.Song.Link == "" || betterRank(src.ProviderRank, dst.ProviderRank)) {
		dst.Song.Link = src.Song.Link
	}
	if src.ProviderRank > 0 && (dst.ProviderRank == 0 || src.ProviderRank < dst.ProviderRank) {
		dst.ProviderRank = src.ProviderRank
	}
	if dst.Pagination == nil {
		dst.Pagination = src.Pagination
	}
	mergeProvider(&dst.Provider, src.Provider)
	mergeProvider(&dst.Song.Provider, src.Provider)
}

func betterRank(candidate, existing int) bool {
	return candidate > 0 && (existing == 0 || candidate < existing)
}

func mergeProvider(dst *string, add string) {
	if add == "" {
		return
	}
	if *dst == "" {
		*dst = add
		return
	}
	for _, p := range strings.Split(*dst, "+") {
		if p == add {
			return
		}
	}
	*dst += "+" + add
}

func pickPagination(scored []ScoredResult) *domain.PaginationInfo {
	for _, s := range scored {
		if s.Result.Pagination != nil {
			return s.Result.Pagination
		}
	}
	return nil
}

func isBenignSearchErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
