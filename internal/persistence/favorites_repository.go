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
	if len(songReorders) == 0 {
		return nil
	}

	return fr.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		caseStmt := "CASE id"
		ids := make([]interface{}, len(songReorders))
		params := make([]interface{}, 0, len(songReorders)*2)

		for i, move := range songReorders {
			caseStmt += " WHEN ? THEN ?"
			params = append(params, move.SongId, move.Order)
			ids[i] = move.SongId
		}
		caseStmt += " END"

		query := fmt.Sprintf("UPDATE favorite_songs SET \"order\" = %s WHERE id IN ?", caseStmt)
		params = append(params, ids)

		if err := tx.Exec(query, params...).Error; err != nil {
			return fmt.Errorf("favorites repository: bulk reorder failed: %w", err)
		}

		return nil
	})
}
