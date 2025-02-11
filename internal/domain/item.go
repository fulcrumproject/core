package domain

import (
	"time"

	"gorm.io/datatypes"
)

type Item struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"not null"`
	Description string
	Properties  datatypes.JSONMap
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

type Repository interface {
	Create(item *Item) error
	GetByID(id uint) (*Item, error)
	Update(item *Item) error
	Delete(id uint) error
	List() ([]Item, error)
}
