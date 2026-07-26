package providers

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestYoutubeIDFromMp3musicsFile(t *testing.T) {
	// hex("MI2qLjle5ug|86e7b5c2")
	id, ok := youtubeIDFromMp3musicsFile("/file/4d4932714c6a6c653575677c3836653762356332")
	if !ok || id != "MI2qLjle5ug" {
		t.Fatalf("got %q ok=%v", id, ok)
	}
	id, ok = youtubeIDFromMp3musicsFile("https://mp3musics.pro/file/4d4932714c6a6c653575677c3836653762356332")
	if !ok || id != "MI2qLjle5ug" {
		t.Fatalf("absolute: %q ok=%v", id, ok)
	}
	if _, ok := youtubeIDFromMp3musicsFile("/file/zz"); ok {
		t.Fatal("expected reject")
	}
}

func TestSplitMp3musicsLabel(t *testing.T) {
	a, title := splitMp3musicsLabel("Olson Siu - Eu vin în sat | Official Video")
	if a != "Olson Siu" || title != "Eu vin în sat" {
		t.Fatalf("got %q — %q", a, title)
	}
	a, title = splitMp3musicsLabel("Pasha Parfeni – Ușor cobor la vale (Official Video)")
	if a != "Pasha Parfeni" || !strings.Contains(title, "Ușor") {
		t.Fatalf("en-dash: %q — %q", a, title)
	}
}

func TestMp3musicsParseHits(t *testing.T) {
	html := `
<body data-api-hash="testhash">
<div class="tc">
  <a class="tc-title" title="Olson Siu - Eu vin în sat | Official Video" href="/mp3/x">Olson Siu - Eu vin în sat | Official Video</a>
  <a class="tc-dl" href="/file/4d4932714c6a6c653575677c3836653762356332">MP3</a>
</div>
<div class="tc">
  <a class="tc-title" title="NoFile - Track" href="/mp3/y">NoFile - Track</a>
</div>
</body>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	hits := parseMp3musicsHits(doc)
	if len(hits) != 1 {
		t.Fatalf("want 1 hit, got %d", len(hits))
	}
	if hits[0].artist != "Olson Siu" || hits[0].title != "Eu vin în sat" || hits[0].ytID != "MI2qLjle5ug" {
		t.Fatalf("%+v", hits[0])
	}
	if hits[0].fileURL != "https://mp3musics.pro/file/4d4932714c6a6c653575677c3836653762356332" {
		t.Fatalf("fileURL: %q", hits[0].fileURL)
	}
	if path, ok := Mp3musicsFileRef(hits[0].fileURL); !ok || path == "" {
		t.Fatalf("Mp3musicsFileRef failed for %q", hits[0].fileURL)
	}
}
