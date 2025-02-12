package database

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type providerRepository struct {
	db *gorm.DB
}

// NewProviderRepository crea una nuova istanza di ProviderRepository
func NewProviderRepository(db *gorm.DB) domain.ProviderRepository {
	return &providerRepository{db: db}
}

func (r *providerRepository) Create(ctx context.Context, provider *domain.Provider) error {
	if err := provider.Validate(); err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Create(provider)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *providerRepository) Update(ctx context.Context, provider *domain.Provider) error {
	// Prima verifichiamo che il Provider esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.Provider{}, provider.ID).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	if err := provider.Validate(); err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Save(provider)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *providerRepository) Delete(ctx context.Context, id domain.UUID) error {
	// Prima verifichiamo che il Provider esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.Provider{}, id).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).Delete(&domain.Provider{}, id)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *providerRepository) FindByID(ctx context.Context, id domain.UUID) (*domain.Provider, error) {
	var provider domain.Provider
	err := r.db.WithContext(ctx).First(&provider, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &provider, nil
}

func (r *providerRepository) List(ctx context.Context, filters map[string]interface{}) ([]domain.Provider, error) {
	var providers []domain.Provider

	query := r.db.WithContext(ctx)
	for key, value := range filters {
		query = query.Where(key, value)
	}

	if err := query.Find(&providers).Error; err != nil {
		return nil, err
	}

	return providers, nil
}

func (r *providerRepository) FindByCountryCode(ctx context.Context, countryCode string) ([]domain.Provider, error) {
	var providers []domain.Provider

	err := r.db.WithContext(ctx).
		Where("country_code = ?", countryCode).
		Find(&providers).Error
	if err != nil {
		return nil, err
	}

	return providers, nil
}

func (r *providerRepository) UpdateState(ctx context.Context, id domain.UUID, state domain.ProviderState) error {
	// Prima verifichiamo che il Provider esista
	exists := r.db.WithContext(ctx).Select("id").First(&domain.Provider{}, id).Error == nil
	if !exists {
		return domain.ErrNotFound
	}

	result := r.db.WithContext(ctx).
		Model(&domain.Provider{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"state":      state,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if result.Error != nil {
		return result.Error
	}

	return nil
}
