package services

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

// ponytail: prove slow + fast providers both land before rank.
func TestSearchMergesAllProviders(t *testing.T) {
	slow := stubProvider{
		name:     "Mp3pm",
		priority: 8,
		delay:    30 * time.Millisecond,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Bistro", Artist: "Morgenstern", Link: "https://a.mp3"}, Provider: "Mp3pm", ProviderRank: 2},
		},
	}
	fast := stubProvider{
		name:     "Mp3mn",
		priority: 7,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://b.mp3"}, Provider: "Mp3mn", ProviderRank: 1},
		},
	}

	svc := NewSearchService([]ports.IMusicProvider{slow, fast}, domain.DefaultSearchConfig(), 0)
	if svc.searchTimeout != 5*time.Second {
		t.Fatalf("default timeout want 5s, got %v", svc.searchTimeout)
	}
	resp, err := svc.Search(context.Background(), "adele", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Songs) != 2 {
		t.Fatalf("want 2 merged songs, got %d", len(resp.Songs))
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
	svc := NewSearchService([]ports.IMusicProvider{low, high}, domain.DefaultSearchConfig(), time.Second)
	got, err := svc.SearchFirst(context.Background(), "adele hello", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Link != "https://pm.mp3" {
		t.Fatalf("want Mp3pm first-wins, got %+v", got)
	}
}

// Lower priority finishes first — still wait for higher-priority playable hit.
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
	svc := NewSearchService([]ports.IMusicProvider{low, high}, domain.DefaultSearchConfig(), time.Second)
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
	p := countingProvider{
		stubProvider: stubProvider{
			name:     "Mp3pm",
			priority: 8,
			results: []domain.ProviderResult{
				{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://a.mp3"}, Provider: "Mp3pm", ProviderRank: 1},
			},
		},
		hits: &hits,
	}
	svc := NewSearchService([]ports.IMusicProvider{p}, domain.DefaultSearchConfig(), time.Second)
	a, err := svc.Search(context.Background(), "Adele Hello", 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Search(context.Background(), "adele  hello", 1)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("want 1 provider hit, got %d", hits.Load())
	}
	if len(a.Songs) != 1 || len(b.Songs) != 1 || a.Songs[0].Link != b.Songs[0].Link {
		t.Fatalf("cache clone mismatch: %+v vs %+v", a.Songs, b.Songs)
	}
	// Mutating one response must not poison the cache.
	a.Songs[0].Image = "https://poison"
	c, err := svc.Search(context.Background(), "adele hello", 1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Songs[0].Image == "https://poison" {
		t.Fatal("cache was mutated via returned slice")
	}
}

func TestSearchCollectEarlyWhenMaxResults(t *testing.T) {
	cfg := domain.DefaultSearchConfig()
	cfg.MaxResults = 2
	fast := stubProvider{
		name:     "Mp3pm",
		priority: 8,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "A", Artist: "X", Link: "https://a.mp3"}, Provider: "Mp3pm", ProviderRank: 1},
			{Song: domain.Song{Title: "B", Artist: "Y", Link: "https://b.mp3"}, Provider: "Mp3pm", ProviderRank: 2},
		},
	}
	slow := stubProvider{
		name:     "Mp3mn",
		priority: 7,
		delay:    200 * time.Millisecond,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "C", Artist: "Z", Link: "https://c.mp3"}, Provider: "Mp3mn", ProviderRank: 1},
		},
	}
	svc := NewSearchService([]ports.IMusicProvider{fast, slow}, cfg, time.Second)
	start := time.Now()
	resp, err := svc.Search(context.Background(), "test", 1)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 150*time.Millisecond {
		t.Fatalf("should early-finish once MaxResults playable, took %v", time.Since(start))
	}
	if len(resp.Songs) != 2 {
		t.Fatalf("want 2 songs, got %d", len(resp.Songs))
	}
}

func TestSearchDedupesSameTrackAcrossProviders(t *testing.T) {
	dup := []domain.ProviderResult{
		{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://slow.mp3"}, Provider: "Mp3mn", ProviderRank: 3},
		{Song: domain.Song{Title: "Hello", Artist: "Adele", Image: "https://img.jpg", Link: "https://fast.mp3"}, Provider: "Mp3pm", ProviderRank: 1},
	}
	got := dedupe(dup)
	if len(got) != 1 {
		t.Fatalf("want 1 merged row, got %d", len(got))
	}
	if got[0].Song.Image != "https://img.jpg" {
		t.Fatalf("image: %q", got[0].Song.Image)
	}
	if got[0].Song.Link != "https://fast.mp3" {
		t.Fatalf("link from better rank: %q", got[0].Song.Link)
	}
	if got[0].Provider != "Mp3mn+Mp3pm" && got[0].Provider != "Mp3pm+Mp3mn" {
		t.Fatalf("combined provider: %q", got[0].Provider)
	}
}

func TestMergeDuplicateFillsGaps(t *testing.T) {
	a := domain.ProviderResult{
		Song:         domain.Song{Title: "X", Artist: "Y", Link: "https://a.mp3"},
		Provider:     "Mp3mn",
		ProviderRank: 3,
	}
	b := domain.ProviderResult{
		Song:         domain.Song{Title: "X", Artist: "Y", Image: "https://img.jpg", Link: "https://b.mp3"},
		Provider:     "Mp3pm",
		ProviderRank: 1,
	}
	mergeDuplicate(&a, &b)
	if a.Song.Image != "https://img.jpg" {
		t.Fatalf("image not merged: %q", a.Song.Image)
	}
	if a.Song.Link != "https://b.mp3" {
		t.Fatalf("better-rank link: %q", a.Song.Link)
	}
	if a.ProviderRank != 1 {
		t.Fatalf("rank not improved: %d", a.ProviderRank)
	}
	if a.Provider != "Mp3mn+Mp3pm" {
		t.Fatalf("provider: %q", a.Provider)
	}
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

type countingProvider struct {
	stubProvider
	hits *atomic.Int32
}

func (c countingProvider) SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error) {
	c.hits.Add(1)
	return c.stubProvider.SearchWithPage(ctx, query, page)
}

var _ ports.IMusicProvider = stubProvider{}
var _ ports.IMusicProvider = countingProvider{}
