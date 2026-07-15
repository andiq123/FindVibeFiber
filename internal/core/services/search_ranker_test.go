package services

import (
	"testing"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

func TestRankResultsExactTitleBeatsRemix(t *testing.T) {
	ranker := NewSearchRanker(domain.DefaultSearchConfig().RankingWeights)
	results := []domain.ProviderResult{
		{Song: domain.Song{Title: "Hello (Remix)", Artist: "Adele"}, Provider: "MuzJam", ProviderRank: 1},
		{Song: domain.Song{Title: "Hello", Artist: "Adele"}, Provider: "Mp3mn", ProviderRank: 2},
	}

	scored := ranker.RankResults(results, "adele hello", map[string]int{"MuzJam": 8, "Mp3mn": 7})
	if len(scored) != 2 {
		t.Fatalf("got %d results", len(scored))
	}
	if scored[0].Result.Song.Title != "Hello" {
		t.Fatalf("want exact title first, got %q", scored[0].Result.Song.Title)
	}
}

func TestRankResultsQueryMatchBeatsProviderOrder(t *testing.T) {
	ranker := NewSearchRanker(domain.DefaultSearchConfig().RankingWeights)
	results := []domain.ProviderResult{
		{Song: domain.Song{Title: "God's Plan", Artist: "Drake"}, Provider: "Mp3mn", ProviderRank: 1},
		{Song: domain.Song{Title: "Hello", Artist: "Adele"}, Provider: "MuzJam", ProviderRank: 5},
		{Song: domain.Song{Title: "One Dance", Artist: "Drake"}, Provider: "Mp3mn", ProviderRank: 2},
	}

	scored := ranker.RankResults(results, "adele hello", map[string]int{"MuzJam": 8, "Mp3mn": 7})
	if scored[0].Result.Song.Title != "Hello" || scored[0].Result.Song.Artist != "Adele" {
		t.Fatalf("want Adele Hello first, got %q — %q (score=%v)",
			scored[0].Result.Song.Artist, scored[0].Result.Song.Title, scored[0].FinalScore)
	}
}
