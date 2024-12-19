package ports

import "github.com/gofiber/fiber/v3"

type IHealthHandlers interface {
	GetHealth(c fiber.Ctx) error
}
