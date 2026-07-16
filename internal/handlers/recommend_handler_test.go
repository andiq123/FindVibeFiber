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

func TestParseChartTitle(t *testing.T) {
	cases := []struct {
		raw, artist, title string
	}{
		{"VANILLA x ALEX VELEA - 7 din 7 (VIDEOCLIP OFICIAL)", "VANILLA", "7 din 7"},
		{"Denis Ramniceanu x BABASHA - Vara nu e vara | Official video", "Denis Ramniceanu", "Vara nu e vara"},
		{"IAN x AZTECA - FOCU'", "IAN", "FOCU'"},
		{"MIRA - Caramela | Official Video", "MIRA", "Caramela"},
	}
	for _, tc := range cases {
		a, title, ok := parseChartTitle(tc.raw)
		if !ok || a != tc.artist || title != tc.title {
			t.Fatalf("%q → (%q, %q, %v) want (%q, %q)", tc.raw, a, title, ok, tc.artist, tc.title)
		}
	}
}

func TestParseKworbMusicPairsUsesMusicTable(t *testing.T) {
	html := `
<div class="overall"><table><tbody>
<tr><td class="text"><div><a href="#">Gaming video not music</a></div></td></tr>
</tbody></table></div>
<div class="music" style="display: none;"><table id="trendingcountry"><tbody>
<tr><td class="text"><div><a href="#">VANILLA x ALEX VELEA - 7 din 7 | Official</a></div></td></tr>
<tr><td class="text"><div><a href="#">MIRA - Caramela | Official Video</a></div></td></tr>
</tbody></table></div>`
	got := parseKworbMusicPairs(html)
	if len(got) != 2 || got[0].artist != "VANILLA" || got[1].title != "Caramela" {
		t.Fatalf("got %+v", got)
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

func TestExploreCacheRoundTrip(t *testing.T) {
	h := &RecommendHandler{}
	if _, ok := h.exploreSnap(); ok {
		t.Fatal("empty cache should miss")
	}
	h.exploreStore([]ExploreSection{{
		ID: "romania", Title: "Romania",
		Songs: []domain.Song{{Title: "A", Artist: "B", Link: "https://x.mp3"}},
	}})
	got, ok := h.exploreSnap()
	if !ok || len(got) != 1 || got[0].ID != "romania" || len(got[0].Songs) != 1 {
		t.Fatalf("got %+v ok=%v", got, ok)
	}
}
