package middleware

import (
	"github.com/andiq123/FindVibeFiber/internal/core/constants"
	"github.com/gofiber/fiber/v3"
)

func RequestSizeLimit() fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Request().Header.ContentLength() > constants.MaxRequestSize {
			return c.Status(413).JSON(fiber.Map{
				"error": "request body too large",
			})
		}
		return c.Next()
	}
}
