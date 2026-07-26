package services

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/constants"
	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/utils"
)

type SearchService struct {
	providers     []ports.IMusicProvider
	priorities    map[string]int
	ranker        *SearchRanker
	config        *domain.SearchConfig
	searchTimeout time.Duration
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

	// Year tokens only apply on Musify — skip unfiltered sources so results stay honest.
	providers := ss.providers
	if parsed.HasYearFilter() {
		providers = yearFilterProviders(ss.providers)
		if len(providers) == 0 {
			return domain.NewSearchResponse([]domain.Song{}, nil), nil
		}
	}

	// ponytail: one /search — fetch every provider, merge, dedupe, rank (no first-wins race).
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

	// Never return empty / non-https links — iOS AVPlayer cannot play them.
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

// SearchFirst walks providers by priority and returns as soon as one yields playable songs.
// Used by explore/radio resolve — faster than waiting on a full multi-provider merge.
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

	for _, p := range providers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pctx, cancel := context.WithTimeout(ctx, ss.searchTimeout)
		got, err := p.SearchWithPage(pctx, text, 1)
		cancel()
		if err != nil {
			utils.GetLogger().Warn("provider search-first failed", "provider", p.Name(), "query", text, "error", err)
			continue
		}
		if len(got) == 0 {
			continue
		}
		songs := playableSongs(got, limit)
		if len(songs) > 0 {
			return songs, nil
		}
	}
	return nil, nil
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

// collect waits for every provider in parallel; each is cancelled after searchTimeout (default 2s).
func (ss *SearchService) collect(ctx context.Context, q ParsedSearchQuery, page int, providers []ports.IMusicProvider) []domain.ProviderResult {
	n := len(providers)
	ch := make(chan []domain.ProviderResult, n)

	for _, p := range providers {
		go func(p ports.IMusicProvider) {
			pctx, cancel := context.WithTimeout(ctx, ss.searchTimeout)
			defer cancel()

			got, err := searchProvider(pctx, p, q, page)
			if err != nil {
				// timeout / cancel → skip that source; others still merge
				utils.GetLogger().Warn("provider search failed", "provider", p.Name(), "query", q.Text, "error", err)
				ch <- nil
				return
			}
			ch <- got
		}(p)
	}

	out := make([]domain.ProviderResult, 0, 40*n)
	for range n {
		if got := <-ch; len(got) > 0 {
			out = append(out, got...)
		}
	}
	return out
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
