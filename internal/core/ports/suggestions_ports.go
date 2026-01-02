package ports

import (
	"context"
)

type ISuggestionsService interface {
	GetSuggestions(ctx context.Context, query string) ([]string, error)
}
