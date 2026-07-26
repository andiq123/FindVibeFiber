package providers

import (
	"testing"
)

func TestIsBlockedBody(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{`<div id="results" class="item">`, false},
		{`<html>Just a moment...</html>`, true},
		{`cf-browser-verification`, true},
		{`DDoS-Guard`, true},
		{``, false},
	}
	for _, tc := range cases {
		if got := isBlockedBody([]byte(tc.body)); got != tc.want {
			t.Fatalf("body %q: got %v want %v", tc.body, got, tc.want)
		}
	}
}

func TestIsBlockedStatus(t *testing.T) {
	if !isBlockedStatus(403) || !isBlockedStatus(429) || isBlockedStatus(200) || isBlockedStatus(404) {
		t.Fatal("status mapping wrong")
	}
}
