package ports

import (
	"context"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/gofiber/fiber/v3"
)

type IFavoritesService interface {
	GetFavorites(ctx context.Context, userId string) ([]domain.FavoriteSong, error)
	AddFavorite(ctx context.Context, userId string, song domain.FavoriteSong) error
	DeleteFavorite(ctx context.Context, songId string) error
	ReorderFavorites(ctx context.Context, songReorders []domain.ReorderRequest) error
}

type IFavoritesRepository interface {
	GetFavorites(ctx context.Context, userId string) ([]domain.FavoriteSong, error)
	AddFavorite(ctx context.Context, userId string, song domain.FavoriteSong) error
	DeleteFavorite(ctx context.Context, songId string) error
	ReorderFavorites(ctx context.Context, songReorders []domain.ReorderRequest) error
}

type IFavoritesHandler interface {
	GetFavorites(c fiber.Ctx) error
	AddFavorite(c fiber.Ctx) error
	DeleteFavorite(c fiber.Ctx) error
	ReorderFavorites(c fiber.Ctx) error
}
