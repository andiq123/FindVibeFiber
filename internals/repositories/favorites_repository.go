package repositories

import (
	"errors"

	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/andiq123/FindVibeFiber/internals/core/models"
	"github.com/andiq123/FindVibeFiber/internals/core/ports"
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

func (f *FavoritesRepository) AddFavorite(userId string, song domain.FavoriteSong) error {
	result := f.DB.First(&domain.FavoriteSong{}, "id = ? AND user_uuid = ?", song.Id, userId)

	if result.Error == nil {
		return errors.New("song already added")
	}

	return f.DB.Create(&song).Error
}

func (f *FavoritesRepository) DeleteFavorite(songId string) error {
	return f.DB.Delete(&domain.FavoriteSong{}, "id = ?", songId).Error
}

func (f *FavoritesRepository) GetFavorites(userId string) ([]domain.FavoriteSong, error) {
	var songs []domain.FavoriteSong
	return songs, f.DB.Where("user_uuid = ?", userId).Find(&songs).Error
}

func (f *FavoritesRepository) ReorderFavorites(songReorders []models.ReorderRequest) error {
	return f.DB.Transaction(func(tx *gorm.DB) error {
		for _, songReorder := range songReorders {
			result := tx.Model(&domain.FavoriteSong{}).Where("id = ?", songReorder.SongId).Update("order", songReorder.Order)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("no rows updated, invalid ID or no change")
			}
		}
		return nil
	})
}
