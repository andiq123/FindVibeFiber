package ports

import (
	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/gofiber/fiber/v3"
)

type IAuthService interface {
	AuthenticateUser(username string) (*domain.User, error)
}

type IAuthRepository interface {
	AuthenticateUser(username string) (*domain.User, error)
	GetUserById(id string) (*domain.User, error)
}

type IAuthHandlers interface {
	AuthenticateUser(c fiber.Ctx) error
}
