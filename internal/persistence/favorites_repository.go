package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/core/ports"
	"gorm.io/gorm"
)

type FavoritesRepository struct {
	DB *gorm.DB
}

var _ ports.IFavoritesRepository = (*FavoritesRepository)(nil)

func NewFavoritesRepository(db *gorm.DB) *FavoritesRepository {
	return &FavoritesRepository{
		DB: db,
	}
}

func (fr *FavoritesRepository) AddFavorite(ctx context.Context, userId string, song domain.FavoriteSong) error {
	var existing domain.FavoriteSong
	err := fr.DB.WithContext(ctx).First(&existing, "id = ? AND user_uuid = ?", song.Id, userId).Error
	if err == nil {
		return domain.ErrAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("favorites repository: check existing failed: %w", err)
	}

	if err := fr.DB.WithContext(ctx).Create(&song).Error; err != nil {
		return fmt.Errorf("favorites repository: create failed: %w", err)
	}
	return nil
}

func (fr *FavoritesRepository) DeleteFavorite(ctx context.Context, songId string) error {
	if err := fr.DB.WithContext(ctx).Delete(&domain.FavoriteSong{}, "id = ?", songId).Error; err != nil {
		return fmt.Errorf("favorites repository: delete failed: %w", err)
	}
	return nil
}

func (fr *FavoritesRepository) GetFavorites(ctx context.Context, userId string) ([]domain.FavoriteSong, error) {
	var songs []domain.FavoriteSong
	if err := fr.DB.WithContext(ctx).Where("user_uuid = ?", userId).Find(&songs).Error; err != nil {
		return nil, fmt.Errorf("favorites repository: find failed: %w", err)
	}
	return songs, nil
}

func (fr *FavoritesRepository) ReorderFavorites(ctx context.Context, songReorders []domain.ReorderRequest) error {
	return fr.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, move := range songReorders {
			result := tx.Model(&domain.FavoriteSong{}).
				Where("id = ?", move.SongId).
				Update("order", move.Order)

			if err := result.Error; err != nil {
				return fmt.Errorf("favorites repository: update order failed for %s: %w", move.SongId, err)
			}
			if result.RowsAffected == 0 {
				return domain.ErrNotFound
			}
		}
		return nil
	})
}
