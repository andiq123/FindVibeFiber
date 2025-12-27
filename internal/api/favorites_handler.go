package api

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/gofiber/fiber/v3"
)

type FavoritesHandler struct {
	favoritesService ports.IFavoritesService
}

var _ ports.IFavoritesHandler = (*FavoritesHandler)(nil)

func NewFavoritesHandler(favoritesService ports.IFavoritesService) *FavoritesHandler {
	return &FavoritesHandler{
		favoritesService: favoritesService,
	}
}

func (fh *FavoritesHandler) AddFavorite(c fiber.Ctx) error {
	userId := c.Params("userId")
	var song domain.FavoriteSong

	if err := c.Bind().JSON(&song); err != nil {
		return err
	}

	err := fh.favoritesService.AddFavorite(c.Context(), userId, song)
	if err != nil {
		return HandleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(map[string]string{"message": "song added"})
}

func (fh *FavoritesHandler) DeleteFavorite(c fiber.Ctx) error {
	songId := c.Params("songId")

	err := fh.favoritesService.DeleteFavorite(c.Context(), songId)
	if err != nil {
		return HandleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(map[string]string{"message": "song deleted"})
}

func (fh *FavoritesHandler) GetFavorites(c fiber.Ctx) error {
	userId := c.Params("userId")

	favorites, err := fh.favoritesService.GetFavorites(c.Context(), userId)
	if err != nil {
		return HandleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(favorites)
}

func (fh *FavoritesHandler) ReorderFavorites(c fiber.Ctx) error {
	var songReorders []domain.ReorderRequest

	if err := c.Bind().JSON(&songReorders); err != nil {
		return err
	}

	err := fh.favoritesService.ReorderFavorites(c.Context(), songReorders)
	if err != nil {
		return HandleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(map[string]string{"message": "songs reordered"})
}
