package handlers

import (
	"errors"
	"net/http"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/gofiber/fiber/v3"
)

func HandleError(c fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	statusCode := http.StatusInternalServerError

	switch {
	case errors.Is(err, domain.ErrNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyExists):
		statusCode = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidInput):
		statusCode = http.StatusBadRequest
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"error": err.Error(),
	})
}
