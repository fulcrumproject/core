package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormConfigPoolValueRepository struct {
	*GormRepository[domain.ConfigPoolValue]
}

var applyConfigPoolValueFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":             StringContainsInsensitiveFilterFieldApplier("name"),
	"configPoolId":     ParserInFilterFieldApplier("config_pool_id", properties.ParseUUID),
	"agentId":          ParserInFilterFieldApplier("agent_id", properties.ParseUUID),
	"infrastructureId": ParserInFilterFieldApplier("infrastructure_id", properties.ParseUUID),
})

var applyConfigPoolValueSort = MapSortApplier(map[string]string{
	"name":      "name",
	"createdAt": "created_at",
})

func NewConfigPoolValueRepository(db *gorm.DB) *GormConfigPoolValueRepository {
	return &GormConfigPoolValueRepository{
		GormRepository: NewGormRepository[domain.ConfigPoolValue](
			db,
			applyConfigPoolValueFilter,
			applyConfigPoolValueSort,
			participantAuthzFilterApplier,
			[]string{"ConfigPool", "Agent", "Infrastructure"},
			[]string{"ConfigPool", "Agent", "Infrastructure"},
		),
	}
}

func (r *GormConfigPoolValueRepository) Update(ctx context.Context, value *domain.ConfigPoolValue) error {
	return r.Save(ctx, value)
}

func (r *GormConfigPoolValueRepository) FindAvailable(ctx context.Context, poolID properties.UUID) ([]*domain.ConfigPoolValue, error) {
	var values []*domain.ConfigPoolValue
	result := r.db.WithContext(ctx).Where("config_pool_id = ? AND agent_id IS NULL AND infrastructure_id IS NULL", poolID).Order("name ASC").Find(&values)
	return values, result.Error
}

func (r *GormConfigPoolValueRepository) FindByAgent(ctx context.Context, agentID properties.UUID) ([]*domain.ConfigPoolValue, error) {
	var values []*domain.ConfigPoolValue
	result := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Order("name ASC").Find(&values)
	return values, result.Error
}

func (r *GormConfigPoolValueRepository) FindByInfrastructure(ctx context.Context, infrastructureID properties.UUID) ([]*domain.ConfigPoolValue, error) {
	var values []*domain.ConfigPoolValue
	result := r.db.WithContext(ctx).Where("infrastructure_id = ?", infrastructureID).Order("name ASC").Find(&values)
	return values, result.Error
}

func (r *GormConfigPoolValueRepository) CountByPool(ctx context.Context, poolID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.ConfigPoolValue{}).Where("config_pool_id = ?", poolID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return count, nil
}

// AuthScope returns the authorization scope for a config pool value. Values inherit
// participant ownership from their parent pool at creation time; global pools (nil
// participant_id) yield AdminOnlyObjectScope so participants can't write to globals.
func (r *GormConfigPoolValueRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	scope, err := r.AuthScopeByFields(ctx, id, "participant_id", "null", "null", "null")
	if err != nil {
		return nil, err
	}
	if d, ok := scope.(*authz.DefaultObjectScope); ok && d.ParticipantID == nil {
		return authz.AdminOnlyObjectScope{}, nil
	}
	return scope, nil
}
