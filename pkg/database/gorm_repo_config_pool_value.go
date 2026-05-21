package database

import (
	"context"
	"errors"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormConfigPoolValueRepository struct {
	*GormRepository[domain.ConfigPoolValue]
}

var applyConfigPoolValueFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":         StringContainsInsensitiveFilterFieldApplier("name"),
	"configPoolId": ParserInFilterFieldApplier("config_pool_id", properties.ParseUUID),
	"agentId":      ParserInFilterFieldApplier("agent_id", properties.ParseUUID),
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
			nil,
			[]string{"ConfigPool", "Agent"},
			[]string{"ConfigPool", "Agent"},
		),
	}
}

func (r *GormConfigPoolValueRepository) Update(ctx context.Context, value *domain.ConfigPoolValue) error {
	return r.Save(ctx, value)
}

func (r *GormConfigPoolValueRepository) FindAvailable(ctx context.Context, poolID properties.UUID) ([]*domain.ConfigPoolValue, error) {
	var values []*domain.ConfigPoolValue
	result := r.db.WithContext(ctx).Where("config_pool_id = ? AND agent_id IS NULL", poolID).Order("name ASC").Find(&values)
	return values, result.Error
}

func (r *GormConfigPoolValueRepository) FindByAgent(ctx context.Context, agentID properties.UUID) ([]*domain.ConfigPoolValue, error) {
	var values []*domain.ConfigPoolValue
	result := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Order("name ASC").Find(&values)
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

// AuthScope returns the authorization scope for a config pool value, inherited from
// its parent config_pool's participant_id. Global pools (nil participant_id) become
// AdminOnlyObjectScope so participants can't write to globals.
func (r *GormConfigPoolValueRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	var result struct {
		ParticipantID *properties.UUID `gorm:"column:participant_id"`
	}

	err := r.db.WithContext(ctx).
		Table("config_pool_values").
		Select("config_pools.participant_id").
		Joins("JOIN config_pools ON config_pools.id = config_pool_values.config_pool_id").
		Where("config_pool_values.id = ?", id).
		Scan(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	if result.ParticipantID == nil {
		return authz.AdminOnlyObjectScope{}, nil
	}
	return &authz.DefaultObjectScope{ParticipantID: result.ParticipantID}, nil
}
