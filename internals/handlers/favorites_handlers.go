package handlers

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/gofiber/fiber/v3"
)

type FavoritesHandlers struct {
	favoritesService ports.IFavoritesService
}

var _ ports.IFavoritesHandler = (*FavoritesHandlers)(nil)

func NewFavoritesHandlers(favoritesService ports.IFavoritesService) *FavoritesHandlers {
	return &FavoritesHandlers{
		favoritesService: favoritesService,
	}
}

func (f *FavoritesHandlers) AddFavorite(c fiber.Ctx) error {
	userId := c.Params("userId")
	var song domain.FavoriteSong

	if err := c.Bind().JSON(&song); err != nil {
		return err
	}

	err := f.favoritesService.AddFavorite(userId, song)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]string{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(map[string]string{"message": "song added"})
}

func (f *FavoritesHandlers) DeleteFavorite(c fiber.Ctx) error {
	songId := c.Params("songId")

	err := f.favoritesService.DeleteFavorite(songId)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]string{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(map[string]string{"message": "song deleted"})
}

func (f *FavoritesHandlers) GetFavorites(c fiber.Ctx) error {
	userId := c.Params("userId")

	favorites, err := f.favoritesService.GetFavorites(userId)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]string{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(favorites)
}

func (f *FavoritesHandlers) ReorderFavorites(c fiber.Ctx) error {
	var songReorders []models.ReorderRequest

	if err := c.Bind().JSON(&songReorders); err != nil {
		return err
	}

	err := f.favoritesService.ReorderFavorites(songReorders)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]string{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(map[string]string{"message": "songs reordered"})
}
