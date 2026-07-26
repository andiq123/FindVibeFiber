package ports

import (
	"context"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

type ISearchService interface {
	Search(ctx context.Context, query string, page int) (*domain.SearchResponse, error)
	// SearchFirst tries providers by priority until one returns playable songs (explore/radio resolve).
	SearchFirst(ctx context.Context, query string, limit int) ([]domain.Song, error)
}
