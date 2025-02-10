package database

import (
	"fmt"

	"fulcrumproject.org/core/internal/domain"
	"gorm.io/gorm"
)

type ItemRepository struct {
	db *gorm.DB
}

func NewItemRepository(db *gorm.DB) *ItemRepository {
	return &ItemRepository{
		db: db,
	}
}

func (r *ItemRepository) Create(item *domain.Item) error {
	result := r.db.Create(item)
	if result.Error != nil {
		return fmt.Errorf("failed to create item: %w", result.Error)
	}
	return nil
}

func (r *ItemRepository) GetByID(id uint) (*domain.Item, error) {
	var item domain.Item
	result := r.db.First(&item, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get item: %w", result.Error)
	}
	return &item, nil
}

func (r *ItemRepository) Update(item *domain.Item) error {
	result := r.db.Save(item)
	if result.Error != nil {
		return fmt.Errorf("failed to update item: %w", result.Error)
	}
	return nil
}

func (r *ItemRepository) Delete(id uint) error {
	result := r.db.Delete(&domain.Item{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete item: %w", result.Error)
	}
	return nil
}

func (r *ItemRepository) List() ([]domain.Item, error) {
	var items []domain.Item
	result := r.db.Find(&items)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list items: %w", result.Error)
	}
	return items, nil
}
