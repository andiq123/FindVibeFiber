package ports

import (
	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/gofiber/fiber/v3"
)

type IMusicFinderService interface {
	FindMusic(query string) ([]models.Song, error)
}

type IMusicFinderHandlers interface {
	FindMusic(c fiber.Ctx) error
}
