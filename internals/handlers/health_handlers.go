package handlers

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/gofiber/fiber/v3"
)

type HealthHandlers struct{}

var _ ports.IHealthHandlers = (*HealthHandlers)(nil)

func NewHealthHandlers() *HealthHandlers {
	return &HealthHandlers{}
}

func (h *HealthHandlers) GetHealth(c fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON("Pong")
}
