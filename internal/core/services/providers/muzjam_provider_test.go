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
}

func TestMuzJamLiveSearch(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("set LIVE=1 to hit muzjam.org")
	}
	res, err := NewMuzJamProvider(&http.Client{Timeout: 15 * time.Second}).
		SearchWithPage(context.Background(), "drake", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("no results")
	}
	if img := res[0].Song.Image; img != "" && !strings.Contains(img, "jpg.muzjam.org") {
		t.Fatalf("image host: %q", img)
	}
}
