package ports

import (
	"context"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

type IFavoritesService interface {
	GetFavorites(ctx context.Context, userId string) ([]domain.FavoriteSong, error)
	AddFavorite(ctx context.Context, userId string, song domain.FavoriteSong) error
	DeleteFavorite(ctx context.Context, songId string) error
	ReorderFavorites(ctx context.Context, songReorders []domain.ReorderRequest) error
	UpdateFavoriteImage(ctx context.Context, songId, image string) error
}

type IFavoritesRepository interface {
	GetFavorites(ctx context.Context, userId string) ([]domain.FavoriteSong, error)
	AddFavorite(ctx context.Context, userId string, song domain.FavoriteSong) error
	DeleteFavorite(ctx context.Context, songId string) error
	ReorderFavorites(ctx context.Context, songReorders []domain.ReorderRequest) error
	UpdateFavoriteImage(ctx context.Context, songId, image string) error
}
