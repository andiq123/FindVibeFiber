package api

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/gofiber/fiber/v3"
)

type HealthHandler struct{}

var _ ports.IHealthHandler = (*HealthHandler)(nil)

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (hh *HealthHandler) GetHealth(c fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON("Pong")
}
