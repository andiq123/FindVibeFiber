package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type stubCatalog struct {
	hits    []CatalogHit
	artists []CatalogArtist
	albums  []domain.ArtistAlbum
	pag     *domain.PaginationInfo
	err     error
}

func (s stubCatalog) Configured() bool { return true }
func (s stubCatalog) Search(context.Context, string, int, int) (CatalogPage, error) {
	return CatalogPage{Hits: s.hits, Artists: s.artists, Pagination: s.pag}, s.err
}
func (s stubCatalog) TopAlbums(context.Context, string, int) ([]domain.ArtistAlbum, error) {
	return s.albums, nil
}

func TestSearchMapsCatalogThroughProviders(t *testing.T) {
	catalog := stubCatalog{
		hits: []CatalogHit{
			{Artist: "Adele", Title: "Hello"},
			{Artist: "Missing", Title: "Nowhere"},
		},
		pag: &domain.PaginationInfo{CurrentPage: 1, HasNextPage: true, TotalPages: 3, TotalResults: 40},
	}
	provider := stubProvider{
		name:     "Mp3pm",
		priority: 8,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://ok.mp3"}, Provider: "Mp3pm", ProviderRank: 1},
		},
	}
	// Second SearchFirst call (Missing Nowhere) returns empty via empty results for other queries —
	// stub always returns Hello; PickPlayableSong will reject Missing Nowhere.
	svc := NewSearchService([]ports.IMusicProvider{provider}, domain.DefaultSearchConfig(), time.Second, catalog)
	resp, err := svc.Search(context.Background(), "adele", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Songs) != 1 || resp.Songs[0].Link != "https://ok.mp3" {
		t.Fatalf("want only mapped Hello, got %+v", resp.Songs)
	}
	if resp.Pagination == nil || !resp.Pagination.HasNextPage {
		t.Fatal("want lastfm pagination preserved")
	}
}

func TestSearchRequiresCatalog(t *testing.T) {
	svc := NewSearchService(nil, domain.DefaultSearchConfig(), time.Second, nil)
	_, err := svc.Search(context.Background(), "adele", 1)
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}

func TestSearchFirstReturnsHighestPriorityProvider(t *testing.T) {
	high := stubProvider{
		name:     "Mp3pm",
		priority: 8,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://pm.mp3"}, Provider: "Mp3pm", ProviderRank: 1},
		},
	}
	low := stubProvider{
		name:     "Mp3mn",
		priority: 7,
		delay:    50 * time.Millisecond,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://mn.mp3"}, Provider: "Mp3mn", ProviderRank: 1},
		},
	}
	svc := NewSearchService([]ports.IMusicProvider{low, high}, domain.DefaultSearchConfig(), time.Second, stubCatalog{})
	got, err := svc.SearchFirst(context.Background(), "adele hello", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Link != "https://pm.mp3" {
		t.Fatalf("want Mp3pm first-wins, got %+v", got)
	}
}

func TestSearchFirstWaitsForHigherPriority(t *testing.T) {
	high := stubProvider{
		name:     "Mp3pm",
		priority: 8,
		delay:    40 * time.Millisecond,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://pm.mp3"}, Provider: "Mp3pm", ProviderRank: 1},
		},
	}
	low := stubProvider{
		name:     "Mp3mn",
		priority: 7,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://mn.mp3"}, Provider: "Mp3mn", ProviderRank: 1},
		},
	}
	svc := NewSearchService([]ports.IMusicProvider{low, high}, domain.DefaultSearchConfig(), time.Second, stubCatalog{})
	start := time.Now()
	got, err := svc.SearchFirst(context.Background(), "adele hello", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Link != "https://pm.mp3" {
		t.Fatalf("want higher-priority link, got %+v", got)
	}
	if time.Since(start) < 35*time.Millisecond {
		t.Fatalf("returned before higher-priority provider could finish")
	}
}

func TestSearchCachesIdenticalQuery(t *testing.T) {
	var hits atomic.Int32
	catalog := &countingCatalog{
		stubCatalog: stubCatalog{
			hits: []CatalogHit{{Artist: "Adele", Title: "Hello"}},
		},
		hits: &hits,
	}
	provider := stubProvider{
		name:     "Mp3pm",
		priority: 8,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://a.mp3"}, Provider: "Mp3pm", ProviderRank: 1},
		},
	}
	svc := NewSearchService([]ports.IMusicProvider{provider}, domain.DefaultSearchConfig(), time.Second, catalog)
	a, err := svc.Search(context.Background(), "Adele Hello", 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Search(context.Background(), "adele  hello", 1)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("want 1 catalog hit, got %d", hits.Load())
	}
	if len(a.Songs) != 1 || len(b.Songs) != 1 || a.Songs[0].Link != b.Songs[0].Link {
		t.Fatalf("cache clone mismatch: %+v vs %+v", a.Songs, b.Songs)
	}
	a.Songs[0].Image = "https://poison"
	c, err := svc.Search(context.Background(), "adele hello", 1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Songs[0].Image == "https://poison" {
		t.Fatal("cache was mutated via returned slice")
	}
}

type countingCatalog struct {
	stubCatalog
	hits *atomic.Int32
}

func (c *countingCatalog) Search(ctx context.Context, query string, page, limit int) (CatalogPage, error) {
	c.hits.Add(1)
	return c.stubCatalog.Search(ctx, query, page, limit)
}

type stubProvider struct {
	name     string
	priority int
	delay    time.Duration
	results  []domain.ProviderResult
	err      error
}

func (s stubProvider) Name() string  { return s.name }
func (s stubProvider) Priority() int { return s.priority }
func (s stubProvider) SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.results, nil
}

var _ ports.IMusicProvider = stubProvider{}
