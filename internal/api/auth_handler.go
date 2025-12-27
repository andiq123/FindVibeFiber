package api

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	authService ports.IAuthService
}

var _ ports.IAuthHandler = (*AuthHandler)(nil)

func NewAuthHandler(authService ports.IAuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (ah *AuthHandler) AuthenticateUser(c fiber.Ctx) error {
	username := c.Params("username")

	user, err := ah.authService.AuthenticateUser(c.Context(), username)

	if err != nil {
		return HandleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(user)
}
