package services

import (
	"testing"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

func TestUpdateFavoriteImageRejectsBadURL(t *testing.T) {
	fs := &FavoritesService{}
	for _, img := range []string{"", "http://x", "ftp://x", string(make([]byte, 1001))} {
		if err := fs.UpdateFavoriteImage(t.Context(), "id", img); err != domain.ErrInvalidInput {
			t.Fatalf("image %q: got %v want ErrInvalidInput", truncate(img), err)
		}
	}
}

func truncate(s string) string {
	if len(s) > 32 {
		return s[:32] + "…"
	}
	return s
}
