package ports

import (
	"context"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

type IMusicProvider interface {
	Name() string
	Search(ctx context.Context, query string) ([]domain.ProviderResult, error)
	Priority() int
}
