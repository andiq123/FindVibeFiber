package services

import (
	"testing"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

func TestRankResultsExactTitleBeatsRemix(t *testing.T) {
	ranker := NewSearchRanker(domain.DefaultSearchConfig().RankingWeights)
	results := []domain.ProviderResult{
		{Song: domain.Song{Title: "Hello (Remix)", Artist: "Adele"}, Provider: "OtherSrc", ProviderRank: 1},
		{Song: domain.Song{Title: "Hello", Artist: "Adele"}, Provider: "Mp3mn", ProviderRank: 2},
	}

	scored := ranker.RankResults(results, "adele hello", map[string]int{"OtherSrc": 8, "Mp3mn": 7})
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
		{Song: domain.Song{Title: "Hello", Artist: "Adele"}, Provider: "OtherSrc", ProviderRank: 5},
		{Song: domain.Song{Title: "One Dance", Artist: "Drake"}, Provider: "Mp3mn", ProviderRank: 2},
	}

	scored := ranker.RankResults(results, "adele hello", map[string]int{"OtherSrc": 8, "Mp3mn": 7})
	if scored[0].Result.Song.Title != "Hello" || scored[0].Result.Song.Artist != "Adele" {
		t.Fatalf("want Adele Hello first, got %q — %q (score=%v)",
			scored[0].Result.Song.Artist, scored[0].Result.Song.Title, scored[0].FinalScore)
	}
}

func TestRankResultsMultiWordQueryPrefersFullMatch(t *testing.T) {
	ranker := NewSearchRanker(domain.DefaultSearchConfig().RankingWeights)
	results := []domain.ProviderResult{
		{Song: domain.Song{Title: "Morgenstern", Artist: "Morgenstern"}, Provider: "Mp3mn", ProviderRank: 1},
		{Song: domain.Song{Title: "Bistro", Artist: "SLAVA MARLOW, MORGENSHTERN"}, Provider: "Mp3mn", ProviderRank: 2},
		{Song: domain.Song{Title: "Morgenstern", Artist: "Rammstein"}, Provider: "Mp3mn", ProviderRank: 3},
	}

	scored := ranker.RankResults(results, "morgenstern bistro", map[string]int{"Mp3mn": 7})
	if scored[0].Result.Song.Title != "Bistro" {
		t.Fatalf("want Bistro top, got %q — %q",
			scored[0].Result.Song.Title, scored[0].Result.Song.Artist)
	}
}

func TestWordMatchesTransliteration(t *testing.T) {
	if !wordMatches("morgenstern", "morgenshtern") {
		t.Fatal("morgenstern should fuzzy-match morgenshtern")
	}
	if wordMatches("bistro", "rammstein") {
		t.Fatal("unrelated words should not match")
	}
}
