package repositories

import (
	"errors"
	"fmt"

	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
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

func (a *AuthRepository) AuthenticateUser(username string) (*domain.User, error) {
	var user domain.User

	result := a.DB.First(&user, "name = ?", username)

	if result.Error != nil {
		user = *domain.NewUser(username)
		resultCreatedUser := a.DB.Create(user)

		if resultCreatedUser.Error != nil {
			fmt.Println("Error creating user: ", resultCreatedUser.Error)
			return nil, result.Error
		}

		return &user, nil
	}

	return &user, nil
}

func (a *AuthRepository) GetUserById(id string) (*domain.User, error) {
	var user domain.User

	result := a.DB.First(&user, "id = ?", id)

	if result.Error != nil {
		return nil, errors.New("user not found")
	}

	return &user, nil
}
