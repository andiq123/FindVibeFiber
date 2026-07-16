package handlers

import "testing"

func TestPickLyricsPrefersPlain(t *testing.T) {
	got := pickLyrics([]lrclibHit{
		{Instrumental: true, PlainLyrics: "nope"},
		{SyncedLyrics: "[00:01.00] Hello\n[00:02.00] World"},
		{PlainLyrics: "Plain text"},
	})
	if got != "Plain text" {
		t.Fatalf("got %q", got)
	}
}

func TestPickLyricsStripsSyncedStamps(t *testing.T) {
	got := pickLyrics([]lrclibHit{
		{SyncedLyrics: "[00:01.00] Hello\n[00:02.00] World\n"},
	})
	if got != "Hello\nWorld" {
		t.Fatalf("got %q", got)
	}
}

func TestAllInstrumental(t *testing.T) {
	if !allInstrumental([]lrclibHit{{Instrumental: true}, {Instrumental: true}}) {
		t.Fatal("want true")
	}
	if allInstrumental(nil) || allInstrumental([]lrclibHit{{Instrumental: false}}) {
		t.Fatal("want false")
	}
}
