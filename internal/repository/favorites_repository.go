package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"gorm.io/gorm"
)

type FavoritesRepository struct {
	DB *gorm.DB
}

func NewFavoritesRepository(db *gorm.DB) *FavoritesRepository {
	return &FavoritesRepository{
		DB: db,
	}
}

func (fr *FavoritesRepository) AddFavorite(ctx context.Context, userId string, song domain.FavoriteSong) error {
	var existing domain.FavoriteSong
	err := fr.DB.WithContext(ctx).Where("id = ? AND user_uuid = ?", song.ID, userId).Take(&existing).Error

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
	err := fr.DB.WithContext(ctx).Where("user_uuid = ?", userId).Order("\"order\" ASC").Find(&songs).Error
	if err != nil {
		return nil, fmt.Errorf("favorites repository: find failed: %w", err)
	}
	return songs, nil
}

func (fr *FavoritesRepository) ReorderFavorites(ctx context.Context, songReorders []domain.ReorderRequest) error {
	if len(songReorders) == 0 {
		return nil
	}

	return fr.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, reorder := range songReorders {
			if err := tx.Model(&domain.FavoriteSong{}).
				Where("id = ?", reorder.SongId).
				Update("order", reorder.Order).Error; err != nil {
				return fmt.Errorf("favorites repository: reorder failed for song %s: %w", reorder.SongId, err)
			}
		}
		return nil
	})
}

func (fr *FavoritesRepository) UpdateFavoriteImage(ctx context.Context, songId, image string) error {
	// ponytail: don't use RowsAffected — Postgres reports 0 when value unchanged.
	var n int64
	if err := fr.DB.WithContext(ctx).Model(&domain.FavoriteSong{}).
		Where("id = ?", songId).Count(&n).Error; err != nil {
		return fmt.Errorf("favorites repository: update image lookup: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	if err := fr.DB.WithContext(ctx).Model(&domain.FavoriteSong{}).
		Where("id = ?", songId).
		Update("image", image).Error; err != nil {
		return fmt.Errorf("favorites repository: update image failed: %w", err)
	}
	return nil
}
