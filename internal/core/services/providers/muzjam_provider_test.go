package providers

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAbsoluteURL(t *testing.T) {
	if got := absoluteURL("//jpg.muzjam.org/img/x.jpg"); got != "https://jpg.muzjam.org/img/x.jpg" {
		t.Fatalf("protocol-relative: %q", got)
	}
	if got := absoluteURL("https://song.muzvibe.org/a.mp3"); got != "https://song.muzvibe.org/a.mp3" {
		t.Fatalf("absolute: %q", got)
	}
	if got := absoluteURL(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
}

func TestMuzJamSearchConstants(t *testing.T) {
	if muzjamSearch != "https://muzjam.org/search/" {
		t.Fatalf("search URL: %q", muzjamSearch)
	}
}

// Live scrape check — set LIVE=1 (muzjam.org must be reachable).
func TestMuzJamLiveSearch(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("set LIVE=1 to hit muzjam.org")
	}
	p := NewMuzJamProvider(&http.Client{Timeout: 15 * time.Second})
	res, err := p.SearchWithPage(context.Background(), "drake", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("no results")
	}
	img := res[0].Song.Image
	link := res[0].Song.Link
	if img != "" && !strings.Contains(img, "jpg.muzjam.org") {
		t.Fatalf("expected jpg.muzjam.org image, got %q", img)
	}
	if link != "" && !strings.HasPrefix(link, "https://") {
		t.Fatalf("expected https link, got %q", link)
	}
}
