package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormServiceOptionTypeRepository struct {
	*GormRepository[domain.ServiceOptionType]
}

var applyServiceOptionTypeFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name": StringInFilterFieldApplier("name"),
	"type": StringInFilterFieldApplier("type"),
})

var applyServiceOptionTypeSort = MapSortApplier(map[string]string{
	"name": "name",
	"type": "type",
})

// NewServiceOptionTypeRepository creates a new instance of ServiceOptionTypeRepository
func NewServiceOptionTypeRepository(db *gorm.DB) *GormServiceOptionTypeRepository {
	repo := &GormServiceOptionTypeRepository{
		GormRepository: NewGormRepository[domain.ServiceOptionType](
			db,
			applyServiceOptionTypeFilter,
			applyServiceOptionTypeSort,
			nil,        // No authz filters
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// FindByType retrieves a service option type by type
func (r *GormServiceOptionTypeRepository) FindByType(ctx context.Context, typeStr string) (*domain.ServiceOptionType, error) {
	var entity domain.ServiceOptionType
	result := r.db.WithContext(ctx).Where("type = ?", typeStr).First(&entity)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: result.Error}
		}
		return nil, result.Error
	}
	return &entity, nil
}

func (r *GormServiceOptionTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	// Service option types are global resources
	return &auth.AllwaysMatchObjectScope{}, nil
}
