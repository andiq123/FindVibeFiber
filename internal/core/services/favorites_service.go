package services

import (
	"context"
	"fmt"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
)

type FavoritesService struct {
	favoritesRepository ports.IFavoritesRepository
	authRepository      ports.IAuthRepository
}

func NewFavoritesService(favoritesRepository ports.IFavoritesRepository, authRepository ports.IAuthRepository) *FavoritesService {
	return &FavoritesService{
		favoritesRepository: favoritesRepository,
		authRepository:      authRepository,
	}
}

func (fs *FavoritesService) AddFavorite(ctx context.Context, userId string, song domain.FavoriteSong) error {
	user, err := fs.authRepository.GetUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("add favorite: %w", err)
	}
	song.UserID = user.ID
	if err := fs.favoritesRepository.AddFavorite(ctx, userId, song); err != nil {
		return fmt.Errorf("add favorite: %w", err)
	}
	return nil
}

func (fs *FavoritesService) DeleteFavorite(ctx context.Context, songId string) error {
	if err := fs.favoritesRepository.DeleteFavorite(ctx, songId); err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}
	return nil
}

func (fs *FavoritesService) GetFavorites(ctx context.Context, userId string) ([]domain.FavoriteSong, error) {
	songs, err := fs.favoritesRepository.GetFavorites(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("get favorites: %w", err)
	}
	return songs, nil
}

func (fs *FavoritesService) ReorderFavorites(ctx context.Context, songReorders []domain.ReorderRequest) error {
	if err := fs.favoritesRepository.ReorderFavorites(ctx, songReorders); err != nil {
		return fmt.Errorf("reorder favorites: %w", err)
	}
	return nil
}
