package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormAgentPoolValueRepository struct {
	*GormRepository[domain.AgentPoolValue]
}

var applyAgentPoolValueFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":        StringContainsInsensitiveFilterFieldApplier("name"),
	"agentPoolId": ParserInFilterFieldApplier("agent_pool_id", properties.ParseUUID),
	"agentId":     ParserInFilterFieldApplier("agent_id", properties.ParseUUID),
})

var applyAgentPoolValueSort = MapSortApplier(map[string]string{
	"name":      "name",
	"createdAt": "created_at",
})

func NewAgentPoolValueRepository(db *gorm.DB) *GormAgentPoolValueRepository {
	return &GormAgentPoolValueRepository{
		GormRepository: NewGormRepository[domain.AgentPoolValue](
			db,
			applyAgentPoolValueFilter,
			applyAgentPoolValueSort,
			nil,
			[]string{"AgentPool", "Agent"},
			[]string{"AgentPool", "Agent"},
		),
	}
}

func (r *GormAgentPoolValueRepository) Update(ctx context.Context, value *domain.AgentPoolValue) error {
	return r.Save(ctx, value)
}

func (r *GormAgentPoolValueRepository) FindAvailable(ctx context.Context, poolID properties.UUID) ([]*domain.AgentPoolValue, error) {
	var values []*domain.AgentPoolValue
	result := r.db.WithContext(ctx).Where("agent_pool_id = ? AND agent_id IS NULL", poolID).Order("name ASC").Find(&values)
	return values, result.Error
}

func (r *GormAgentPoolValueRepository) FindByAgent(ctx context.Context, agentID properties.UUID) ([]*domain.AgentPoolValue, error) {
	var values []*domain.AgentPoolValue
	result := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Order("name ASC").Find(&values)
	return values, result.Error
}

func (r *GormAgentPoolValueRepository) CountByPool(ctx context.Context, poolID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.AgentPoolValue{}).Where("agent_pool_id = ?", poolID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

func (r *GormAgentPoolValueRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return &authz.AllwaysMatchObjectScope{}, nil
}
