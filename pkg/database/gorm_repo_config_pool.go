package database

import (
	"context"
	"errors"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GormConfigPoolRepository struct {
	*GormRepository[domain.ConfigPool]
}

var applyConfigPoolFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name":          StringContainsInsensitiveFilterFieldApplier("name"),
	"type":          StringInFilterFieldApplier("type"),
	"generatorType": StringInFilterFieldApplier("generator_type"),
	"participantId": ParserInFilterFieldApplier("participant_id", properties.ParseUUID),
})

var applyConfigPoolSort = MapSortApplier(map[string]string{
	"name":      "name",
	"type":      "type",
	"createdAt": "created_at",
})

func NewConfigPoolRepository(db *gorm.DB) *GormConfigPoolRepository {
	return &GormConfigPoolRepository{
		GormRepository: NewGormRepository[domain.ConfigPool](
			db,
			applyConfigPoolFilter,
			applyConfigPoolSort,
			participantAuthzFilterApplier,
			[]string{"Participant"},
			[]string{"Participant"},
		),
	}
}

func (r *GormConfigPoolRepository) Update(ctx context.Context, pool *domain.ConfigPool) error {
	return r.Save(ctx, pool)
}

func (r *GormConfigPoolRepository) FindByTypeAndProvider(ctx context.Context, poolType string, providerID *properties.UUID) (*domain.ConfigPool, error) {
	q := r.db.WithContext(ctx).Where("type = ?", poolType)
	if providerID != nil {
		q = q.Where("participant_id IS NULL OR participant_id = ?", *providerID)
	}

	var entity domain.ConfigPool
	if err := q.First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundErrorf("no config pool with type %s for provider", poolType)
		}
		return nil, err
	}
	return &entity, nil
}

func (r *GormConfigPoolRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	scope, err := r.AuthScopeByFields(ctx, id, "participant_id", "null", "null", "null")
	if err != nil {
		return nil, err
	}
	if d, ok := scope.(*authz.DefaultObjectScope); ok && d.ParticipantID == nil {
		return authz.AdminOnlyObjectScope{}, nil
	}
	return scope, nil
}
