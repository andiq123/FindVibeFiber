package api

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/gofiber/fiber/v3"
)

type MusicFinderHandler struct {
	musicFinderService ports.IMusicFinderService
}

var _ ports.IMusicFinderHandler = (*MusicFinderHandler)(nil)

func NewMusicFinderHandler(musicFinderService ports.IMusicFinderService) *MusicFinderHandler {
	return &MusicFinderHandler{musicFinderService: musicFinderService}
}

func (mfh *MusicFinderHandler) FindMusic(c fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(http.StatusBadRequest).JSON(map[string]string{"error": "query parameter 'q' is required"})
	}

	songs, err := mfh.musicFinderService.FindMusic(c.Context(), query)
	if err != nil {
		return HandleError(c, err)
	}

	if len(songs) == 0 {
		return c.Status(http.StatusNotFound).JSON(map[string]string{"error": "no songs found"})
	}

	return c.Status(http.StatusOK).JSON(songs)
}
