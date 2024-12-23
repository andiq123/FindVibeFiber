package handlers

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/gofiber/fiber/v3"
)

type MusicFinderHandlers struct {
	musicFinderService ports.IMusicFinderService
}

var _ ports.IMusicFinderHandlers = (*MusicFinderHandlers)(nil)

func NewMusicFinderHandlers(musicFinderService ports.IMusicFinderService) *MusicFinderHandlers {
	return &MusicFinderHandlers{musicFinderService: musicFinderService}
}

func (m *MusicFinderHandlers) FindMusic(c fiber.Ctx) error {
	query := c.Query("q")

	resultChan := make(chan []models.Song)
	errorChan := make(chan error)
	doneChan := make(chan struct{})

	go func() {
		defer close(resultChan)
		defer close(errorChan)
		songs, err := m.musicFinderService.FindMusic(query)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- songs
	}()

	var songs []models.Song
	select {
	case res := <-resultChan:
		songs = res
	case err := <-errorChan:
		return c.Status(http.StatusInternalServerError).JSON(map[string]string{"error": err.Error()})
	case <-doneChan:
		return c.Status(http.StatusInternalServerError).JSON(map[string]string{"error": "task cancelled"})
	}

	if len(songs) == 0 {
		return c.Status(http.StatusNotFound).JSON(map[string]string{"error": "no songs found"})
	}

	return c.Status(http.StatusOK).JSON(songs)
}