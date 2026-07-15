package services

import (
	"testing"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

func TestRankResultsExactTitleBeatsRemix(t *testing.T) {
	ranker := NewSearchRanker(domain.DefaultSearchConfig().RankingWeights)
	results := []domain.ProviderResult{
		{Song: domain.Song{Title: "Hello (Remix)", Artist: "Adele"}, ProviderRank: 1},
		{Song: domain.Song{Title: "Hello", Artist: "Adele"}, ProviderRank: 2},
	}

	scored := ranker.RankResults(results, "adele hello", 8)
	if len(scored) != 2 {
		t.Fatalf("got %d results", len(scored))
	}
	if scored[0].Result.Song.Title != "Hello" {
		t.Fatalf("want exact title first, got %q", scored[0].Result.Song.Title)
	}
}
