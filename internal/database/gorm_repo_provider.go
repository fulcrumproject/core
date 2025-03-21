package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormProviderRepository struct {
	*GormRepository[domain.Provider]
}

var applyProviderFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name":        stringInFilterFieldApplier("name"),
	"state":       parserInFilterFieldApplier("state", domain.ParseProviderState),
	"countryCode": parserInFilterFieldApplier("country_code", domain.ParseCountryCode),
})

var applyProviderSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewProviderRepository creates a new instance of ProviderRepository
func NewProviderRepository(db *gorm.DB) *GormProviderRepository {
	repo := &GormProviderRepository{
		GormRepository: NewGormRepository[domain.Provider](
			db,
			applyProviderFilter,
			applyProviderSort,
			nil,        // No authz filters
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// AuthScope returns the auth scope for the provider
func (r *GormProviderRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	return r.getAuthScope(ctx, id, "id as provider_id")
}
