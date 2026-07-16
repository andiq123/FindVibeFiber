package services

import (
	"context"
	"testing"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

func TestCoverCacheHitSkipsNetwork(t *testing.T) {
	cs := NewCoverService(nil) // nil client panics only on miss
	cs.put("adele hello", "https://img.example/a.jpg")
	got := cs.Lookup(context.Background(), "  Adele   Hello ")
	if got != "https://img.example/a.jpg" {
		t.Fatalf("got %q", got)
	}
}

func TestCoverFillSongs(t *testing.T) {
	cs := NewCoverService(nil)
	cs.put("adele hello", "https://img.example/a.jpg")
	songs := []domain.Song{
		{Title: "Hello", Artist: "Adele"},
		{Title: "Has", Artist: "Art", Image: "https://keep.me"},
	}
	cs.FillSongs(context.Background(), songs)
	if songs[0].Image != "https://img.example/a.jpg" {
		t.Fatalf("filled: %q", songs[0].Image)
	}
	if songs[1].Image != "https://keep.me" {
		t.Fatalf("kept: %q", songs[1].Image)
	}
}
