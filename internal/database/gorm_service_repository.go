package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type serviceRepository struct {
	db *gorm.DB
}

// NewServiceRepository creates a new instance of ServiceRepository
func NewServiceRepository(db *gorm.DB) domain.ServiceRepository {
	return &serviceRepository{db: db}
}

func (r *serviceRepository) Create(ctx context.Context, service *domain.Service) error {
	result := r.db.WithContext(ctx).Create(service)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceRepository) Save(ctx context.Context, service *domain.Service) error {
	result := r.db.WithContext(ctx).Save(service)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceRepository) Delete(ctx context.Context, id domain.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.Service{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *serviceRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	var service domain.Service

	err := r.db.WithContext(ctx).
		Preload("Agent").
		Preload("ServiceType").
		Preload("Group").
		First(&service, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &service, nil
}

var serviceFilterConfigs = map[string]FilterConfig{
	"name":  {},
	"state": {},
}

func (r *serviceRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.Service], error) {
	var services []domain.Service
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&domain.Service{}).
		Preload("Agent").
		Preload("ServiceType").
		Preload("Group")

	query, totalItems, err := applyFindAndCount(query, filter, serviceFilterConfigs, sorting, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&services).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(services, totalItems, pagination), nil
}

func (r *serviceRepository) Count(ctx context.Context, filter *domain.SimpleFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Service{})
	_, count, err := applyFilterAndCount(query, filter, serviceFilterConfigs)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *serviceRepository) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Service{}).
		Where("group_id = ?", groupID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
