package providers

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestAbsoluteURL(t *testing.T) {
	if got := absoluteURL("//jpg.muzjam.org/img/x.jpg"); got != "https://jpg.muzjam.org/img/x.jpg" {
		t.Fatalf("protocol-relative: %q", got)
	}
	if got := absoluteURL("https://song.muzvibe.org/a.mp3"); got != "https://song.muzvibe.org/a.mp3" {
		t.Fatalf("absolute: %q", got)
	}
	if got := absoluteURL("http://song.muzvibe.org/a.mp3"); got != "https://song.muzvibe.org/a.mp3" {
		t.Fatalf("http upgrade: %q", got)
	}
}

func TestPlayableLink(t *testing.T) {
	if got := playableLink("//song.muzvibe.org/a.mp3"); got != "https://song.muzvibe.org/a.mp3" {
		t.Fatalf("got %q", got)
	}
	if playableLink("") != "" || playableLink("javascript:void(0)") != "" {
		t.Fatal("expected empty for bad hrefs")
	}
}

func TestMuzJamParseSkipsEmptyLink(t *testing.T) {
	html := `<div id="results">
		<div class="item"><div class="title">A</div><div class="artist"><a>Art</a></div><a class="link" href=""></a></div>
		<div class="item"><div class="title">B</div><div class="artist"><a>Art</a></div><a class="link" href="//song.muzvibe.org/b.mp3"></a></div>
	</div>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	p := NewMuzJamProvider(&http.Client{})
	got := p.parseResults(doc, 1)
	if len(got) != 1 || got[0].Song.Title != "B" {
		t.Fatalf("got %#v", got)
	}
	if got[0].Song.Link != "https://song.muzvibe.org/b.mp3" {
		t.Fatalf("link %q", got[0].Song.Link)
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
