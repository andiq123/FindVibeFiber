package ports

import (
	"context"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

type IMusicProvider interface {
	Name() string
	SearchWithPage(ctx context.Context, query string, page int) ([]domain.ProviderResult, error)
	Priority() int
}

// YearFilterProvider can apply release-year bounds (e.g. Musify yearFrom/yearTo).
type YearFilterProvider interface {
	SearchWithPageYears(ctx context.Context, query string, page, yearFrom, yearTo int) ([]domain.ProviderResult, error)
}
