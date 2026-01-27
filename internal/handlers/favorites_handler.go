package handlers

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/andiq123/FindVibeFiber/internal/utils"
	"github.com/gofiber/fiber/v3"
)

type FavoritesHandler struct {
	favoritesService ports.IFavoritesService
}

func NewFavoritesHandler(favoritesService ports.IFavoritesService) *FavoritesHandler {
	return &FavoritesHandler{
		favoritesService: favoritesService,
	}
}

func (fh *FavoritesHandler) AddFavorite(c fiber.Ctx) error {
	userId := c.Params("userId")
	
	if err := utils.ValidateUserID(userId); err != nil {
		return HandleError(c, err)
	}

	var song domain.FavoriteSong
	if err := c.Bind().JSON(&song); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := fh.favoritesService.AddFavorite(c.Context(), userId, song); err != nil {
		return HandleError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "song added"})
}

func (fh *FavoritesHandler) DeleteFavorite(c fiber.Ctx) error {
	songId := c.Params("songId")

	if err := utils.ValidateSongID(songId); err != nil {
		return HandleError(c, err)
	}

	if err := fh.favoritesService.DeleteFavorite(c.Context(), songId); err != nil {
		return HandleError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}

func (fh *FavoritesHandler) GetFavorites(c fiber.Ctx) error {
	userId := c.Params("userId")

	if err := utils.ValidateUserID(userId); err != nil {
		return HandleError(c, err)
	}

	favorites, err := fh.favoritesService.GetFavorites(c.Context(), userId)
	if err != nil {
		return HandleError(c, err)
	}

	return c.JSON(favorites)
}

func (fh *FavoritesHandler) ReorderFavorites(c fiber.Ctx) error {
	var songReorders []domain.ReorderRequest

	if err := c.Bind().JSON(&songReorders); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := fh.favoritesService.ReorderFavorites(c.Context(), songReorders); err != nil {
		return HandleError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}
