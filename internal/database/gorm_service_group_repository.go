package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type serviceGroupRepository struct {
	db *gorm.DB
}

// NewServiceGroupRepository creates a new instance of ServiceGroupRepository
func NewServiceGroupRepository(db *gorm.DB) domain.ServiceGroupRepository {
	return &serviceGroupRepository{db: db}
}

func (r *serviceGroupRepository) Create(ctx context.Context, serviceGroup *domain.ServiceGroup) error {
	result := r.db.WithContext(ctx).Create(serviceGroup)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceGroupRepository) Save(ctx context.Context, serviceGroup *domain.ServiceGroup) error {
	result := r.db.WithContext(ctx).Save(serviceGroup)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceGroupRepository) Delete(ctx context.Context, id domain.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.ServiceGroup{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceGroupRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.ServiceGroup, error) {
	var serviceGroup domain.ServiceGroup

	err := r.db.WithContext(ctx).First(&serviceGroup, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &serviceGroup, nil
}

var serviceGroupFilterConfigs = map[string]FilterConfig{
	"name": {},
}

func (r *serviceGroupRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.ServiceGroup], error) {
	var serviceGroups []domain.ServiceGroup
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&domain.ServiceGroup{})
	query, totalItems, err := applyFindAndCount(query, filter, serviceGroupFilterConfigs, sorting, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&serviceGroups).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(serviceGroups, totalItems, pagination), nil
}

func (r *serviceGroupRepository) Count(ctx context.Context, filter *domain.SimpleFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.ServiceGroup{})
	_, count, err := applyFilterAndCount(query, filter, serviceGroupFilterConfigs)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *serviceGroupRepository) CountByService(ctx context.Context, serviceID domain.UUID) (int64, error) {
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
