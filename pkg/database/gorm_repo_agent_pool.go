package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormAgentPoolRepository struct {
	*GormRepository[domain.AgentPool]
}

var applyAgentPoolFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":          StringContainsInsensitiveFilterFieldApplier("name"),
	"type":          StringInFilterFieldApplier("type"),
	"generatorType": StringInFilterFieldApplier("generator_type"),
})

var applyAgentPoolSort = MapSortApplier(map[string]string{
	"name":      "name",
	"type":      "type",
	"createdAt": "created_at",
})

func NewAgentPoolRepository(db *gorm.DB) *GormAgentPoolRepository {
	return &GormAgentPoolRepository{
		GormRepository: NewGormRepository[domain.AgentPool](
			db,
			applyAgentPoolFilter,
			applyAgentPoolSort,
			nil,
			[]string{},
			[]string{},
		),
	}
}

func (r *GormAgentPoolRepository) Update(ctx context.Context, pool *domain.AgentPool) error {
	return r.Save(ctx, pool)
}

func (r *GormAgentPoolRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return &authz.AllwaysMatchObjectScope{}, nil
}
