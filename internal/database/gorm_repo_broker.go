package database

import (
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormBrokerRepository struct {
	*GormRepository[domain.Broker]
}

var applyBrokerFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
})

var applyBrokerSort = mapSortApplier(map[string]string{
	"name": "name",
})

// NewBrokerRepository creates a new instance of BrokerRepository
func NewBrokerRepository(db *gorm.DB) *GormBrokerRepository {
	repo := &GormBrokerRepository{
		GormRepository: NewGormRepository[domain.Broker](
			db,
			applyBrokerFilter,
			applyBrokerSort,
			brokerAuthzFilterApplier, // No authz filters
			[]string{},               // No preload paths needed
			[]string{},               // No preload paths needed
		),
	}
	return repo
}

// brokerAuthzFilterApplier applies authorization scoping to broker queries
func brokerAuthzFilterApplier(s *domain.AuthScope, q *gorm.DB) *gorm.DB {
	if s.BrokerID != nil {
		return q.Where("id = ?", s.BrokerID)
	}
	return q
}
