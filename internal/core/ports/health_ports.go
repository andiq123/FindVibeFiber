package ports

import "github.com/gofiber/fiber/v3"

type IHealthHandler interface {
	GetHealth(c fiber.Ctx) error
}
