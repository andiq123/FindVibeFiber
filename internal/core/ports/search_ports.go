package ports

import (
	"context"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

type ISearchService interface {
	Search(ctx context.Context, query string, page int) (*domain.SearchResponse, error)
}
