package services

import (
	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
)

type FavoritesService struct {
	favoritesRepository ports.IFavoritesRepository
	authRepository      ports.IAuthRepository
}

var _ ports.IFavoritesService = (*FavoritesService)(nil)

func NewFavoritesService(favoritesRepository ports.IFavoritesRepository, authRepository ports.IAuthRepository) *FavoritesService {
	return &FavoritesService{
		favoritesRepository: favoritesRepository,
		authRepository:      authRepository,
	}
}

func (f *FavoritesService) AddFavorite(userId string, song domain.FavoriteSong) error {
	user, err := f.authRepository.GetUserById(userId)
	if err != nil {
		return err
	}
	song.UserID = user.Id
	return f.favoritesRepository.AddFavorite(userId, song)
}

func (f *FavoritesService) DeleteFavorite(songId string) error {
	return f.favoritesRepository.DeleteFavorite(songId)
}

func (f *FavoritesService) GetFavorites(userId string) ([]domain.FavoriteSong, error) {
	songs, err := f.favoritesRepository.GetFavorites(userId)
	if err != nil {
		return nil, err
	}

	return songs, nil
}

func (f *FavoritesService) ReorderFavorites(songReorders []models.ReorderRequest) error {
	return f.favoritesRepository.ReorderFavorites(songReorders)
}
