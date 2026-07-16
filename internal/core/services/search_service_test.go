package services

import (
	"context"
	"testing"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

// ponytail: prove slow + fast providers both land before rank.
func TestSearchMergesAllProviders(t *testing.T) {
	slow := stubProvider{
		name:     "MuzJam",
		priority: 8,
		delay:    30 * time.Millisecond,
		results: []domain.ProviderResult{
			{Song: domain.Song{Title: "Bistro", Artist: "Morgenstern", Link: "https://a.mp3"}, Provider: "MuzJam", ProviderRank: 2},
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
	if svc.searchTimeout != 2*time.Second {
		t.Fatalf("default timeout want 2s, got %v", svc.searchTimeout)
	}
	resp, err := svc.Search(context.Background(), "adele", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Songs) != 2 {
		t.Fatalf("want 2 merged songs, got %d", len(resp.Songs))
	}
}

func TestSearchDedupesSameTrackAcrossProviders(t *testing.T) {
	dup := []domain.ProviderResult{
		{Song: domain.Song{Title: "Hello", Artist: "Adele", Link: "https://slow.mp3"}, Provider: "Mp3mn", ProviderRank: 3},
		{Song: domain.Song{Title: "Hello", Artist: "Adele", Image: "https://img.jpg", Link: "https://fast.mp3"}, Provider: "MuzJam", ProviderRank: 1},
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
	if got[0].Provider != "Mp3mn+MuzJam" && got[0].Provider != "MuzJam+Mp3mn" {
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
		Provider:     "MuzJam",
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
	if a.Provider != "Mp3mn+MuzJam" {
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

var _ ports.IMusicProvider = stubProvider{}
