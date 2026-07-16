package services

import "testing"

func TestLatin1ToUTF8(t *testing.T) {
	// Google sends ã (0xe3) for Romanian ă
	in := []byte("sandu ciorb\xe3")
	got := string(latin1ToUTF8(in))
	want := "sandu ciorbã"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRoSuggestFix(t *testing.T) {
	if got := roSuggestFix.Replace("sandu ciorbã"); got != "sandu ciorbă" {
		t.Fatalf("got %q", got)
	}
}
