package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProbeMusifyMarkersPast64KB(t *testing.T) {
	// Real musify.club pages put track rows after ~75KB of head assets.
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html><head>")
	b.WriteString(strings.Repeat("<!-- pad -->", 7000)) // ~84KB
	b.WriteString(`</head><body>
<div class="tracklist__row playlist__item" data-artist="A" data-name="T">
  <div data-url="/track/pl/1/t.mp3"></div>
</div>
</body></html>`)
	html := b.String()
	if len(html) <= 64<<10 {
		t.Fatalf("fixture too small (%d); need >64KiB to catch the old limit bug", len(html))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") == "" {
			t.Errorf("missing Referer")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	t.Cleanup(srv.Close)

	hh := NewHealthHandler(srv.Client())
	st := hh.probe(sourceSpec{
		Name:    "Musify",
		Host:    "musify.club",
		URL:     srv.URL + "/en/search?searchText=test&type=song",
		Referer: srv.URL + "/en/",
		Markers: [][]byte{
			[]byte(`tracklist__row`),
			[]byte(`/track/pl/`),
		},
	})
	if !st.OK {
		t.Fatalf("expected OK when markers sit past 64KiB")
	}
}

func TestProbeFailsWhenMarkersMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>no tracks</body></html>`))
	}))
	t.Cleanup(srv.Close)

	hh := NewHealthHandler(srv.Client())
	st := hh.probe(sourceSpec{
		Name:    "Musify",
		URL:     srv.URL,
		Markers: [][]byte{[]byte(`tracklist__row`)},
	})
	if st.OK {
		t.Fatal("expected not OK without markers")
	}
}
