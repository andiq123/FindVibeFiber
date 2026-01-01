package services

import (
	"context"
	"sync"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type MusicAggregatorService struct {
	providers []ports.IMusicProvider
	ranker    *ResultRanker
	config    *domain.SearchConfig
}

func NewMusicAggregatorService(providers []ports.IMusicProvider, config *domain.SearchConfig) *MusicAggregatorService {
	if config == nil {
		config = domain.DefaultSearchConfig()
	}

	return &MusicAggregatorService{
		providers: providers,
		ranker:    NewResultRanker(config.RankingWeights),
		config:    config,
	}
}

func (mas *MusicAggregatorService) FindMusic(ctx context.Context, query string) ([]domain.Song, error) {
	if len(mas.providers) == 0 {
		return []domain.Song{}, nil
	}

	var allResults []domain.ProviderResult

	if mas.config.EnableParallelSearch && len(mas.providers) > 1 {
		allResults = mas.searchParallel(ctx, query)
	} else {
		allResults = mas.searchSequential(ctx, query)
	}

	if len(allResults) == 0 {
		return []domain.Song{}, nil
	}

	if mas.config.EnableDeduplication {
		allResults = mas.deduplicateResults(allResults)
	}

	maxPriority := mas.getMaxProviderPriority()
	scoredResults := mas.ranker.RankResults(allResults, query, maxPriority)

	songs := make([]domain.Song, 0, len(scoredResults))
	maxResults := mas.config.MaxResults
	if maxResults <= 0 || maxResults > len(scoredResults) {
		maxResults = len(scoredResults)
	}

	for i := 0; i < maxResults; i++ {
		songs = append(songs, scoredResults[i].Result.Song)
	}

	return songs, nil
}

func (mas *MusicAggregatorService) searchParallel(ctx context.Context, query string) []domain.ProviderResult {
	resultsChan := make(chan []domain.ProviderResult, len(mas.providers))
	var wg sync.WaitGroup

	for _, provider := range mas.providers {
		wg.Add(1)
		go func(p ports.IMusicProvider) {
			defer wg.Done()

			results, err := p.Search(ctx, query)
			if err == nil && len(results) > 0 {
				resultsChan <- results
			}
		}(provider)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	allResults := make([]domain.ProviderResult, 0, len(mas.providers)*10)
	for results := range resultsChan {
		allResults = append(allResults, results...)
	}

	return allResults
}

func (mas *MusicAggregatorService) searchSequential(ctx context.Context, query string) []domain.ProviderResult {
	allResults := make([]domain.ProviderResult, 0, len(mas.providers)*10)

	for _, provider := range mas.providers {
		results, err := provider.Search(ctx, query)

		if err == nil && len(results) > 0 {
			allResults = append(allResults, results...)
		}
	}

	return allResults
}

func (mas *MusicAggregatorService) deduplicateResults(results []domain.ProviderResult) []domain.ProviderResult {
	seen := make(map[string]*domain.ProviderResult)
	deduplicated := make([]domain.ProviderResult, 0, len(results))

	for _, result := range results {
		key := mas.generateDeduplicationKey(result.Song)

		if existing, found := seen[key]; found {
			if mas.shouldReplace(existing, &result) {
				seen[key] = &result
			}
		} else {
			seen[key] = &result
		}
	}

	for _, result := range seen {
		deduplicated = append(deduplicated, *result)
	}

	return deduplicated
}

func (mas *MusicAggregatorService) generateDeduplicationKey(song domain.Song) string {
	normalizedTitle := normalizeString(song.Title)
	normalizedArtist := normalizeString(song.Artist)
	return normalizedTitle + "|" + normalizedArtist
}

func (mas *MusicAggregatorService) shouldReplace(existing, new *domain.ProviderResult) bool {
	if new.MatchScore > existing.MatchScore {
		return true
	}

	if new.MatchScore == existing.MatchScore {
		hasNewMetadata := mas.hasMetadata(new.Song)
		hasExistingMetadata := mas.hasMetadata(existing.Song)

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

func (mas *MusicAggregatorService) hasMetadata(song domain.Song) bool {
	return song.Image != "" && song.Link != ""
}

func (mas *MusicAggregatorService) getMaxProviderPriority() int {
	maxPriority := 0
	for _, provider := range mas.providers {
		if provider.Priority() > maxPriority {
			maxPriority = provider.Priority()
		}
	}
	return maxPriority
}
