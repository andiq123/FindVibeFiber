package handlers

import (
	"net/http"

	"github.com/andiq123/FindVibeFiber/internals/core/ports"
	"github.com/gofiber/fiber/v3"
)

type AuthHandlers struct {
	authService ports.IAuthService
}

var _ ports.IAuthHandlers = (*AuthHandlers)(nil)

func NewAuthHandlers(authService ports.IAuthService) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
	}
}

func (a *AuthHandlers) AuthenticateUser(c fiber.Ctx) error {
	username := c.Params("username")

	userFromDB, err := a.authService.AuthenticateUser(username)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(map[string]string{"error": err.Error()})
	}

	return c.Status(http.StatusOK).JSON(userFromDB)
}
