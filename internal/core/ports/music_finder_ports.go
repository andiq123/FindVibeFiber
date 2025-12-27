package ports

import (
	"context"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/gofiber/fiber/v3"
)

type IMusicFinderService interface {
	FindMusic(ctx context.Context, query string) ([]domain.Song, error)
}

type IMusicFinderHandler interface {
	FindMusic(c fiber.Ctx) error
}
