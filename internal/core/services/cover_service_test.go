package services

import (
	"context"
	"testing"
	"time"

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

func TestCoverMissExpiresSooner(t *testing.T) {
	cs := NewCoverService(nil)
	cs.put("no hit", "")
	e := cs.cache["no hit"]
	if e.url != "" {
		t.Fatalf("want empty miss, got %q", e.url)
	}
	ttl := time.Until(e.exp)
	if ttl > coverMissTTL+time.Second || ttl < coverMissTTL-time.Minute {
		t.Fatalf("miss TTL ~%v, got %v", coverMissTTL, ttl)
	}
	cs.put("hit", "https://img.example/b.jpg")
	hitTTL := time.Until(cs.cache["hit"].exp)
	if hitTTL < coverMissTTL*2 {
		t.Fatalf("hit should use long TTL, got %v", hitTTL)
	}
}
