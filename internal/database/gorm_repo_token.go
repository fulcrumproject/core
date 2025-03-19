package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormTokenRepository struct {
	*GormRepository[domain.Token]
}

var applyTokenFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"name": stringInFilterFieldApplier("name"),
	"role": stringInFilterFieldApplier("role"),
})

var applyTokenSort = mapSortApplier(map[string]string{
	"name":      "name",
	"expireAt":  "expire_at",
	"createdAt": "created_at",
})

// NewTokenRepository creates a new instance of TokenRepository
func NewTokenRepository(db *gorm.DB) *GormTokenRepository {
	repo := &GormTokenRepository{
		GormRepository: NewGormRepository[domain.Token](
			db,
			applyTokenFilter,
			applyTokenSort,
			tokenAuthzFilterApplier,
			[]string{}, // No preload paths needed for finding by ID
			[]string{}, // No preload paths needed for list
		),
	}
	return repo
}

// FindByHashedValue finds a token by its hashed value
func (r *GormTokenRepository) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	var token domain.Token
	err := r.db.WithContext(ctx).Where("hashed_value = ?", hashedValue).First(&token).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &token, nil
}

// DeleteByAgentID removes all tokens associated with an agent ID
func (r *GormTokenRepository) DeleteByAgentID(ctx context.Context, agentID domain.UUID) error {
	// Delete all tokens with the given agent ID
	result := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&domain.Token{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// DeleteByProviderID removes all tokens associated with a provider ID
func (r *GormTokenRepository) DeleteByProviderID(ctx context.Context, providerID domain.UUID) error {
	// Delete all tokens with the given provider ID
	result := r.db.WithContext(ctx).Where("provider_id = ?", providerID).Delete(&domain.Token{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// DeleteByBrokerID removes all tokens associated with a broker ID
func (r *GormTokenRepository) DeleteByBrokerID(ctx context.Context, brokerID domain.UUID) error {
	// Delete all tokens with the given broker ID
	result := r.db.WithContext(ctx).Where("broker_id = ?", brokerID).Delete(&domain.Token{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func tokenAuthzFilterApplier(s *domain.AuthScope, q *gorm.DB) *gorm.DB {
	if s.ProviderID != nil {
		return q.Where("provider_id", s.ProviderID)
	} else if s.BrokerID != nil {
		return q.Where("broker_id", s.BrokerID)
	} else if s.AgentID != nil {
		return q.Where("agent_id", s.AgentID)
	}
	return q
}
