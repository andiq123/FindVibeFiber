package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"gorm.io/gorm"
)

type AuthRepository struct {
	DB *gorm.DB
}

var _ ports.IAuthRepository = (*AuthRepository)(nil)

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{
		DB: db,
	}
}

func (ar *AuthRepository) AuthenticateUser(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User

	if err := ar.DB.WithContext(ctx).First(&user, "name = ?", username).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = *domain.NewUser(username)
			if err := ar.DB.WithContext(ctx).Create(&user).Error; err != nil {
				return nil, fmt.Errorf("auth repository: failed to create user: %w", err)
			}
			return &user, nil
		}
		return nil, fmt.Errorf("auth repository: database error: %w", err)
	}

	return &user, nil
}

func (ar *AuthRepository) GetUserById(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User

	if err := ar.DB.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("auth repository: database error: %w", err)
	}

	return &user, nil
}
