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
			[]string{},
			[]string{},
		),
	}
}

func (r *GormConfigPoolRepository) Update(ctx context.Context, pool *domain.ConfigPool) error {
	return r.Save(ctx, pool)
}

func (r *GormConfigPoolRepository) FindByTypeAndParticipant(ctx context.Context, poolType string, participantID *properties.UUID) (*domain.ConfigPool, error) {
	q := r.db.WithContext(ctx).Where("type = ?", poolType)
	if participantID == nil {
		q = q.Where("participant_id IS NULL")
	} else {
		q = q.Where("participant_id = ?", *participantID)
	}

	var entity domain.ConfigPool
	result := q.First(&entity)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if participantID == nil {
				return nil, domain.NewNotFoundErrorf("global config pool with type %s", poolType)
			}
			return nil, domain.NewNotFoundErrorf("config pool with type %s for participant %s", poolType, *participantID)
		}
		return nil, result.Error
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
