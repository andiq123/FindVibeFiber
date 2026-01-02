package handlers

import "github.com/gofiber/fiber/v3"

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (hh *HealthHandler) GetHealth(c fiber.Ctx) error {
	return c.JSON("Pong")
}
