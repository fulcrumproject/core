package database

import (
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormProviderRepository struct {
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
func NewProviderRepository(db *gorm.DB) domain.ProviderRepository {
	repo := &gormProviderRepository{
		GormRepository: NewGormRepository[domain.Provider](
			db,
			applyProviderFilter,
			applyProviderSort,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}
