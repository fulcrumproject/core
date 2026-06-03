package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormInfrastructureTypeRepository struct {
	*GormRepository[domain.InfrastructureType]
}

var applyInfrastructureTypeFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name": StringContainsInsensitiveFilterFieldApplier("name"),
})

var applyInfrastructureTypeSort = MapSortApplier(map[string]string{
	"name": "name",
})

func NewInfrastructureTypeRepository(db *gorm.DB) *GormInfrastructureTypeRepository {
	return &GormInfrastructureTypeRepository{
		GormRepository: NewGormRepository[domain.InfrastructureType](
			db,
			applyInfrastructureTypeFilter,
			applyInfrastructureTypeSort,
			nil,
			[]string{},
			[]string{},
		),
	}
}

func (r *GormInfrastructureTypeRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return &authz.AllwaysMatchObjectScope{}, nil
}
