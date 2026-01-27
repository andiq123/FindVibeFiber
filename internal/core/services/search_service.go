package services

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/utils"
)

type SearchService struct {
	providers   []ports.IMusicProvider
	ranker      *SearchRanker
	config      *domain.SearchConfig
	searchTimeout time.Duration
}

func NewSearchService(providers []ports.IMusicProvider, config *domain.SearchConfig, timeout time.Duration) *SearchService {
	if config == nil {
		config = domain.DefaultSearchConfig()
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	return &SearchService{
		providers:     providers,
		ranker:        NewSearchRanker(config.RankingWeights),
		config:        config,
		searchTimeout: timeout,
	}
}

func (ss *SearchService) Search(ctx context.Context, query string, page int) (*domain.SearchResponse, error) {
	if len(ss.providers) == 0 {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}

	allResults := ss.searchParallel(ctx, query, page)

	if len(allResults) == 0 {
		return domain.NewSearchResponse([]domain.Song{}, nil), nil
	}

	allResults = ss.deduplicateResults(allResults)

	maxPriority := ss.getMaxProviderPriority()
	scoredResults := ss.ranker.RankResults(allResults, query, maxPriority)

	maxResults := ss.config.MaxResults
	if maxResults <= 0 || maxResults > len(scoredResults) {
		maxResults = len(scoredResults)
	}

	songs := make([]domain.Song, maxResults)
	var pagination *domain.PaginationInfo
	for i := 0; i < maxResults; i++ {
		songs[i] = scoredResults[i].Result.Song
		if pagination == nil && scoredResults[i].Result.Pagination != nil {
			pagination = scoredResults[i].Result.Pagination
		}
	}

	return domain.NewSearchResponse(songs, pagination), nil
}

func (ss *SearchService) searchParallel(ctx context.Context, query string, page int) []domain.ProviderResult {
	timeoutCtx, cancel := context.WithTimeout(ctx, ss.searchTimeout)
	defer cancel()

	resultsChan := make(chan []domain.ProviderResult, len(ss.providers))
	var wg sync.WaitGroup

	for _, provider := range ss.providers {
		wg.Add(1)
		go func(p ports.IMusicProvider) {
			defer wg.Done()

			var results []domain.ProviderResult
			var err error

			if paginatedProvider, ok := p.(interface {
				SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error)
			}); ok {
				results, err = paginatedProvider.SearchWithPage(timeoutCtx, query, page)
			} else {
				results, err = p.Search(timeoutCtx, query)
			}

			if err == nil && len(results) > 0 {
				select {
				case resultsChan <- results:
				case <-timeoutCtx.Done():
					return
				}
			}
		}(provider)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	estimatedCapacity := len(ss.providers) * 40
	if ss.config.MaxResults > 0 && estimatedCapacity > ss.config.MaxResults*2 {
		estimatedCapacity = ss.config.MaxResults * 2
	}
	allResults := make([]domain.ProviderResult, 0, estimatedCapacity)

	for {
		select {
		case results, ok := <-resultsChan:
			if !ok {
				return allResults
			}
			allResults = append(allResults, results...)
		case <-timeoutCtx.Done():
			return allResults
		}
	}
}

func (ss *SearchService) deduplicateResults(results []domain.ProviderResult) []domain.ProviderResult {
	seen := make(map[string]int, len(results)/2)
	deduplicated := make([]domain.ProviderResult, 0, len(results))

	for i := range results {
		key := ss.generateDeduplicationKey(results[i].Song)

		if existingIdx, found := seen[key]; found {
			if ss.shouldReplace(&deduplicated[existingIdx], &results[i]) {
				deduplicated[existingIdx] = results[i]
			}
		} else {
			seen[key] = len(deduplicated)
			deduplicated = append(deduplicated, results[i])
		}
	}

	return deduplicated
}

func (ss *SearchService) generateDeduplicationKey(song domain.Song) string {
	normalizedTitle := utils.NormalizeString(song.Title)
	normalizedArtist := utils.NormalizeString(song.Artist)

	var key strings.Builder
	key.Grow(len(normalizedTitle) + len(normalizedArtist) + 1)
	key.WriteString(normalizedTitle)
	key.WriteByte('|')
	key.WriteString(normalizedArtist)
	return key.String()
}

func (ss *SearchService) shouldReplace(existing, new *domain.ProviderResult) bool {
	if new.MatchScore > existing.MatchScore {
		return true
	}

	if new.MatchScore == existing.MatchScore {
		hasNewMetadata := ss.hasMetadata(new.Song)
		hasExistingMetadata := ss.hasMetadata(existing.Song)

		if hasNewMetadata && !hasExistingMetadata {
			return true
		}

		if !hasNewMetadata && hasExistingMetadata {
			return false
		}

		if new.ProviderRank < existing.ProviderRank {
			return true
		}
	}

	return false
}

func (ss *SearchService) hasMetadata(song domain.Song) bool {
	return song.Image != "" && song.Link != ""
}

func (ss *SearchService) getMaxProviderPriority() int {
	maxPriority := 0
	for _, provider := range ss.providers {
		if provider.Priority() > maxPriority {
			maxPriority = provider.Priority()
		}
	}
	return maxPriority
}
