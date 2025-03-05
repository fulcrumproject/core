package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormServiceGroupRepository struct {
	*GormRepository[domain.ServiceGroup]
}

var applyServiceGroupFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyServiceGroupSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewServiceGroupRepository creates a new instance of ServiceGroupRepository
func NewServiceGroupRepository(db *gorm.DB) domain.ServiceGroupRepository {
	repo := &gormServiceGroupRepository{
		GormRepository: NewGormRepository[domain.ServiceGroup](
			db,
			applyServiceGroupFilter,
			applyServiceGroupSort,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// CountByService returns the number of service groups associated with a specific service
func (r *gormServiceGroupRepository) CountByService(ctx context.Context, serviceID domain.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.ServiceGroup{}).
		Joins("JOIN services ON services.group_id = service_groups.id").
		Where("services.id = ?", serviceID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
