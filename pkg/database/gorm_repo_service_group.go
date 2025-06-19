package database

import (
	"context"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormServiceGroupRepository struct {
	*GormRepository[domain.ServiceGroup]
}

var applyServiceGroupFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyServiceGroupSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewServiceGroupRepository creates a new instance of ServiceGroupRepository
func NewServiceGroupRepository(db *gorm.DB) *GormServiceGroupRepository {
	repo := &GormServiceGroupRepository{
		GormRepository: NewGormRepository[domain.ServiceGroup](
			db,
			applyServiceGroupFilter,
			applyServiceGroupSort,
			serviceGroupAuthzFilterApplier,
			[]string{"Participant"}, // Preload participant for Get
			[]string{"Participant"}, // Preload participant for List
		),
	}
	return repo
}

// CountByService returns the number of service groups associated with a specific service
func (r *GormServiceGroupRepository) CountByService(ctx context.Context, serviceID properties.UUID) (int64, error) {
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

func serviceGroupAuthzFilterApplier(s *auth.IdentityScope, q *gorm.DB) *gorm.DB {
	if s.ParticipantID != nil {
		return q.Where("consumer_id = ?", s.ParticipantID)
	}
	return q
}

func (r *GormServiceGroupRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return r.getAuthScope(ctx, id, "null", "null", "null", "consumer_id")
}
