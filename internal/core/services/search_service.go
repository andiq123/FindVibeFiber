package services

import (
	"context"
	"strings"
	"time"

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
		timeout = time.Second
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

	// ponytail: one /search — fetch every provider, merge, dedupe, rank (no first-wins race).
	results := ss.collect(ctx, query, page)
	if len(results) == 0 {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}

	results = dedupe(results)
	scored := ss.ranker.RankResults(results, query, ss.priorities)

	n := ss.config.MaxResults
	if n <= 0 || n > len(scored) {
		n = len(scored)
	}

	songs := make([]domain.Song, n)
	for i := 0; i < n; i++ {
		songs[i] = scored[i].Result.Song
	}
	return domain.NewSearchResponse(songs, pickPagination(scored)), nil
}

// collect waits for every provider in parallel; each is cancelled after searchTimeout (default 1s).
func (ss *SearchService) collect(ctx context.Context, query string, page int) []domain.ProviderResult {
	n := len(ss.providers)
	ch := make(chan []domain.ProviderResult, n)

	for _, p := range ss.providers {
		go func(p ports.IMusicProvider) {
			pctx, cancel := context.WithTimeout(ctx, ss.searchTimeout)
			defer cancel()

			got, err := p.SearchWithPage(pctx, query, page)
			if err != nil {
				// timeout / cancel → skip that source; others still merge
				utils.GetLogger().Warn("provider search failed", "provider", p.Name(), "query", query, "error", err)
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
