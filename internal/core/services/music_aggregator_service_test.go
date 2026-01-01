package services

import (
	"context"
	"testing"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type MockProvider struct {
	name     string
	priority int
	results  []domain.ProviderResult
	err      error
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Priority() int {
	return m.priority
}

func (m *MockProvider) Search(ctx context.Context, query string) ([]domain.ProviderResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestMusicAggregatorService_FindMusic(t *testing.T) {
	t.Run("Single provider returns ranked results", func(t *testing.T) {
		provider := &MockProvider{
			name:     "TestProvider",
			priority: 8,
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Irrelevant Song", "Artist A", "img1", "link1"),
					"TestProvider", 0.3, 3,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Pizza Tower OST", "Artist B", "img2", "link2"),
					"TestProvider", 1.0, 1,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Pizza Song", "Artist C", "img3", "link3"),
					"TestProvider", 0.6, 2,
				),
			},
		}

		config := domain.DefaultSearchConfig()
		config.MaxResults = 10

		aggregator := NewMusicAggregatorService([]ports.IMusicProvider{provider}, config)
		results, err := aggregator.FindMusic(context.Background(), "pizza tower ost")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}

		if results[0].Title != "Pizza Tower OST" {
			t.Errorf("Expected first result to be 'Pizza Tower OST', got '%s'", results[0].Title)
		}
	})

	t.Run("Deduplication removes duplicate songs", func(t *testing.T) {
		provider1 := &MockProvider{
			name:     "Provider1",
			priority: 8,
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Pizza Tower", "Artist A", "img1", "link1"),
					"Provider1", 0.9, 1,
				),
			},
		}

		provider2 := &MockProvider{
			name:     "Provider2",
			priority: 7,
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Pizza Tower", "Artist A", "img2", "link2"),
					"Provider2", 0.85, 1,
				),
			},
		}

		config := domain.DefaultSearchConfig()
		config.EnableDeduplication = true

		aggregator := NewMusicAggregatorService([]ports.IMusicProvider{provider1, provider2}, config)
		results, err := aggregator.FindMusic(context.Background(), "pizza tower")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 result after deduplication, got %d", len(results))
		}
	})

	t.Run("MaxResults limits returned songs", func(t *testing.T) {
		provider := &MockProvider{
			name:     "TestProvider",
			priority: 8,
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Song 1", "Artist", "img1", "link1"),
					"TestProvider", 0.9, 1,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Song 2", "Artist", "img2", "link2"),
					"TestProvider", 0.8, 2,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Song 3", "Artist", "img3", "link3"),
					"TestProvider", 0.7, 3,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Song 4", "Artist", "img4", "link4"),
					"TestProvider", 0.6, 4,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Song 5", "Artist", "img5", "link5"),
					"TestProvider", 0.5, 5,
				),
			},
		}

		config := domain.DefaultSearchConfig()
		config.MaxResults = 3

		aggregator := NewMusicAggregatorService([]ports.IMusicProvider{provider}, config)
		results, err := aggregator.FindMusic(context.Background(), "song")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results (maxResults), got %d", len(results))
		}
	})

	t.Run("Empty providers returns empty results", func(t *testing.T) {
		config := domain.DefaultSearchConfig()
		aggregator := NewMusicAggregatorService([]ports.IMusicProvider{}, config)
		results, err := aggregator.FindMusic(context.Background(), "test")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 0 {
			t.Fatalf("Expected 0 results, got %d", len(results))
		}
	})

	t.Run("Parallel search merges results from multiple providers", func(t *testing.T) {
		provider1 := &MockProvider{
			name:     "Provider1",
			priority: 8,
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Song A", "Artist 1", "img1", "link1"),
					"Provider1", 0.8, 1,
				),
			},
		}

		provider2 := &MockProvider{
			name:     "Provider2",
			priority: 7,
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Song B", "Artist 2", "img2", "link2"),
					"Provider2", 0.9, 1,
				),
			},
		}

		config := domain.DefaultSearchConfig()
		config.EnableParallelSearch = true

		aggregator := NewMusicAggregatorService([]ports.IMusicProvider{provider1, provider2}, config)
		results, err := aggregator.FindMusic(context.Background(), "song")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 2 {
			t.Fatalf("Expected 2 results from both providers, got %d", len(results))
		}
	})
}

func TestMusicAggregatorService_Deduplication(t *testing.T) {
	config := domain.DefaultSearchConfig()
	aggregator := NewMusicAggregatorService([]ports.IMusicProvider{}, config)

	t.Run("Generates consistent deduplication keys", func(t *testing.T) {
		song1 := *domain.NewSong("Pizza Tower", "Artist A", "img1", "link1")
		song2 := *domain.NewSong("pizza tower", "artist a", "img2", "link2")
		song3 := *domain.NewSong("  PIZZA   TOWER  ", "  ARTIST   A  ", "img3", "link3")

		key1 := aggregator.generateDeduplicationKey(song1)
		key2 := aggregator.generateDeduplicationKey(song2)
		key3 := aggregator.generateDeduplicationKey(song3)

		if key1 != key2 || key2 != key3 {
			t.Errorf("Expected all keys to be equal, got: '%s', '%s', '%s'", key1, key2, key3)
		}
	})

	t.Run("Prefers results with better metadata", func(t *testing.T) {
		result1 := &domain.ProviderResult{
			Song:         *domain.NewSong("Title", "Artist", "", ""),
			MatchScore:   0.8,
			ProviderRank: 1,
		}

		result2 := &domain.ProviderResult{
			Song:         *domain.NewSong("Title", "Artist", "image.jpg", "http://link"),
			MatchScore:   0.8,
			ProviderRank: 2,
		}

		if !aggregator.shouldReplace(result1, result2) {
			t.Error("Should prefer result with metadata over result without")
		}

		if aggregator.shouldReplace(result2, result1) {
			t.Error("Should not replace result with metadata with one without")
		}
	})
}
