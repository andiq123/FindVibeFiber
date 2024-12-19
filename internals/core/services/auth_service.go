package services

import (
	"strings"

	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
)

type AuthService struct {
	authRepository ports.IAuthRepository
}

var _ ports.IAuthService = (*AuthService)(nil)

func NewAuthService(repository ports.IAuthRepository) *AuthService {
	return &AuthService{
		authRepository: repository,
	}
}

func (a *AuthService) AuthenticateUser(username string) (*domain.User, error) {
	username = strings.ToLower(username)

	return a.authRepository.AuthenticateUser(username)
}
