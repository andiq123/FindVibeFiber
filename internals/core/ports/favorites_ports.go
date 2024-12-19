package ports

import (
	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/gofiber/fiber/v3"
)

type IFavoritesService interface {
	GetFavorites(userId string) ([]domain.FavoriteSong, error)
	AddFavorite(userId string, song domain.FavoriteSong) error
	DeleteFavorite(songId string) error
	ReorderFavorites(songReorders []models.ReorderRequest) error
}

type IFavoritesRepository interface {
	GetFavorites(userId string) ([]domain.FavoriteSong, error)
	AddFavorite(userId string, song domain.FavoriteSong) error
	DeleteFavorite(songId string) error
	ReorderFavorites(songReorders []models.ReorderRequest) error
}

type IFavoritesHandler interface {
	GetFavorites(c fiber.Ctx) error
	AddFavorite(c fiber.Ctx) error
	DeleteFavorite(c fiber.Ctx) error
	ReorderFavorites(c fiber.Ctx) error
}
