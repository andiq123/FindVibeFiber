package handlers

import "testing"

func TestUpgradeItunesArtwork(t *testing.T) {
	in := "https://is1-ssl.mzstatic.com/image/thumb/Music115/v4/e8/43/5f/e8435ffa-b6b9-b171-40ab-4ff3959ab661/886443919266.jpg/100x100bb.jpg"
	want := "https://is1-ssl.mzstatic.com/image/thumb/Music115/v4/e8/43/5f/e8435ffa-b6b9-b171-40ab-4ff3959ab661/886443919266.jpg/1000x1000bb.jpg"
	if got := upgradeItunesArtwork(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
