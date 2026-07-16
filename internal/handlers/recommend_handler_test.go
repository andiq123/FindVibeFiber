package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

type stubSearch struct {
	hits map[string][]domain.Song
}

func (s stubSearch) Search(_ context.Context, query string, _ int) (*domain.SearchResponse, error) {
	if songs, ok := s.hits[strings.ToLower(query)]; ok {
		return domain.NewSearchResponse(songs, nil), nil
	}
	return domain.NewSearchResponse(nil, nil), nil
}

func TestCoreTitleStripsRemixVariants(t *testing.T) {
	a := coreTitle("Collide (Extended Mix)")
	b := coreTitle("Collide feat Rosi Golan [Extended Mix]")
	c := coreTitle("Collide (Original Mix)")
	if a != "collide" || b != "collide" || c != "collide" {
		t.Fatalf("got %q %q %q", a, b, c)
	}
}

func TestUniquePairsDropsRemixDupes(t *testing.T) {
	seed := lastfmPair{"Vicetone", "Collide"}
	got := uniquePairs([]lastfmPair{
		{"Vicetone", "Collide (Original Mix)"},
		{"Vicetone", "Collide (Extended Mix)"},
		{"Nero", "Promises"},
		{"Nero", "Promises (Original Mix)"},
	}, seed)
	if len(got) != 1 || got[0].artist != "Nero" {
		t.Fatalf("got %+v", got)
	}
}

func TestArtistCandidatesSplitsCollabs(t *testing.T) {
	got := artistCandidates("Satoshi, magnat, feoctist")
	if len(got) < 3 || got[0] != "Satoshi" {
		t.Fatalf("want Satoshi first, got %v", got)
	}
}

func TestSearchFallbackByArtist(t *testing.T) {
	h := &RecommendHandler{
		search: stubSearch{hits: map[string][]domain.Song{
			"satoshi": {
				{Title: "Pațanii", Artist: "Satoshi, magnat, feoctist", Link: "https://seed.mp3"},
				{Title: "Viva, Moldova!", Artist: "Satoshi", Link: "https://a.mp3"},
				{Title: "Bate, vântule", Artist: "Satoshi, Irina Rimes", Link: "https://b.mp3"},
			},
		}},
	}
	seed := lastfmPair{"Satoshi, magnat, feoctist", "Pațanii"}
	got := h.searchFallback(context.Background(), []string{"Satoshi"}, seed)
	if len(got) != 2 || got[0].Link != "https://a.mp3" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveSkipsSeedAndRemixDupes(t *testing.T) {
	h := &RecommendHandler{
		search: stubSearch{hits: map[string][]domain.Song{
			"vicetone collide": {
				{Title: "Collide (Extended Mix)", Artist: "Vicetone", Link: "https://a.mp3"},
			},
			"nero promises": {
				{Title: "Promises", Artist: "Nero", Link: "https://b.mp3"},
				{Title: "Promises (Remix)", Artist: "Nero", Link: "https://c.mp3"},
			},
			"flux pavilion bass cannon": {
				{Title: "Bass Cannon", Artist: "Flux Pavilion", Link: "https://d.mp3"},
			},
		}},
	}
	seed := lastfmPair{"Vicetone", "Collide"}
	got := h.resolve(context.Background(), []lastfmPair{
		{"Vicetone", "Collide"},
		{"Nero", "Promises"},
		{"Flux Pavilion", "Bass Cannon"},
	}, seed)
	if len(got) != 2 {
		t.Fatalf("want 2 unique other tracks, got %+v", got)
	}
	if got[0].Link != "https://b.mp3" || got[1].Link != "https://d.mp3" {
		t.Fatalf("got %+v", got)
	}
}
