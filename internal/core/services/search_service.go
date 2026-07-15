package services

import (
	"context"
	"sync"
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
		timeout = 3 * time.Second
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

	timeoutCtx, cancel := context.WithTimeout(ctx, ss.searchTimeout)
	defer cancel()

	results := ss.collect(timeoutCtx, query, page)
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
	var pagination *domain.PaginationInfo
	for i := 0; i < n; i++ {
		songs[i] = scored[i].Result.Song
		if pagination == nil {
			pagination = scored[i].Result.Pagination
		}
	}
	return domain.NewSearchResponse(songs, pagination), nil
}

func (ss *SearchService) collect(ctx context.Context, query string, page int) []domain.ProviderResult {
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		out = make([]domain.ProviderResult, 0, 40*len(ss.providers))
	)
	for _, p := range ss.providers {
		wg.Add(1)
		go func(p ports.IMusicProvider) {
			defer wg.Done()
			got, err := p.SearchWithPage(ctx, query, page)
			if err != nil {
				utils.GetLogger().Warn("provider search failed", "provider", p.Name(), "query", query, "error", err)
				return
			}
			if len(got) == 0 {
				return
			}
			mu.Lock()
			out = append(out, got...)
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	return out
}

func dedupe(results []domain.ProviderResult) []domain.ProviderResult {
	seen := make(map[string]int, len(results))
	out := make([]domain.ProviderResult, 0, len(results))

	for i := range results {
		key := utils.NormalizeString(results[i].Song.Title) + "|" + utils.NormalizeString(results[i].Song.Artist)
		if idx, ok := seen[key]; ok {
			if prefer(&out[idx], &results[i]) {
				out[idx] = results[i]
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, results[i])
	}
	return out
}

// prefer keeps the richer duplicate (cover art), else better scrape rank.
func prefer(existing, candidate *domain.ProviderResult) bool {
	exImg, newImg := existing.Song.Image != "", candidate.Song.Image != ""
	if exImg != newImg {
		return newImg
	}
	return candidate.ProviderRank < existing.ProviderRank
}
