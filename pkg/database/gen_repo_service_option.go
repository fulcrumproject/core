// ServiceOption repository implementation using GORM Gen
// Provides type-safe database operations for ServiceOption entities
package database

import (
	"context"
	"encoding/json"
	
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type GenServiceOptionRepository struct {
	q *Query
}

func NewGenServiceOptionRepository(db *gorm.DB) *GenServiceOptionRepository {
	return &GenServiceOptionRepository{q: Use(db)}
}

func (r *GenServiceOptionRepository) Create(ctx context.Context, entity *domain.ServiceOption) error {
	return r.q.ServiceOption.WithContext(ctx).Create(entity)
}

func (r *GenServiceOptionRepository) Save(ctx context.Context, entity *domain.ServiceOption) error {
	result, err := r.q.ServiceOption.WithContext(ctx).Where(r.q.ServiceOption.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceOptionRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.ServiceOption.WithContext(ctx).Where(r.q.ServiceOption.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenServiceOptionRepository) Get(ctx context.Context, id properties.UUID) (*domain.ServiceOption, error) {
	entity, err := r.q.ServiceOption.WithContext(ctx).Where(r.q.ServiceOption.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenServiceOptionRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.ServiceOption.WithContext(ctx).Where(r.q.ServiceOption.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenServiceOptionRepository) Count(ctx context.Context) (int64, error) {
	return r.q.ServiceOption.WithContext(ctx).Count()
}

func (r *GenServiceOptionRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.ServiceOption], error) {
	query := r.q.ServiceOption.WithContext(ctx)
	query = applyGenServiceOptionAuthz(query, scope)

	result, err := PaginateQuery[domain.ServiceOption, IServiceOptionDo](ctx, query, pageReq,
		applyGenServiceOptionFilters,
		applyGenServiceOptionSort,
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ServiceOption, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.ServiceOption]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenServiceOptionRepository) FindByProviderAndTypeAndValue(ctx context.Context, providerID, typeID properties.UUID, value any) (*domain.ServiceOption, error) {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var entity domain.ServiceOption
	err = r.q.ServiceOption.WithContext(ctx).UnderlyingDB().
		Where("provider_id = ?", providerID).
		Where("service_option_type_id = ?", typeID).
		Where("value = ?", valueJSON).
		Where("enabled = ?", true).
		First(&entity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &entity, nil
}

func (r *GenServiceOptionRepository) ListByProvider(ctx context.Context, providerID properties.UUID) ([]*domain.ServiceOption, error) {
	return r.q.ServiceOption.WithContext(ctx).
		Where(r.q.ServiceOption.ProviderID.Eq(providerID)).
		Order(r.q.ServiceOption.DisplayOrder, r.q.ServiceOption.Name).
		Find()
}

func (r *GenServiceOptionRepository) ListByProviderAndType(ctx context.Context, providerID, typeID properties.UUID) ([]*domain.ServiceOption, error) {
	return r.q.ServiceOption.WithContext(ctx).
		Where(r.q.ServiceOption.ProviderID.Eq(providerID)).
		Where(r.q.ServiceOption.ServiceOptionTypeID.Eq(typeID)).
		Order(r.q.ServiceOption.DisplayOrder, r.q.ServiceOption.Name).
		Find()
}

func (r *GenServiceOptionRepository) ListEnabledByProviderAndType(ctx context.Context, providerID, typeID properties.UUID) ([]*domain.ServiceOption, error) {
	return r.q.ServiceOption.WithContext(ctx).
		Where(r.q.ServiceOption.ProviderID.Eq(providerID)).
		Where(r.q.ServiceOption.ServiceOptionTypeID.Eq(typeID)).
		Where(r.q.ServiceOption.Enabled.Is(true)).
		Order(r.q.ServiceOption.DisplayOrder, r.q.ServiceOption.Name).
		Find()
}

func (r *GenServiceOptionRepository) CountByServiceOptionType(ctx context.Context, typeID properties.UUID) (int64, error) {
	return r.q.ServiceOption.WithContext(ctx).
		Where(r.q.ServiceOption.ServiceOptionTypeID.Eq(typeID)).
		Count()
}

func (r *GenServiceOptionRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	entity, err := r.q.ServiceOption.WithContext(ctx).
		Select(r.q.ServiceOption.ProviderID).
		Where(r.q.ServiceOption.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return &auth.DefaultObjectScope{ProviderID: &entity.ProviderID}, nil
}

func applyGenServiceOptionAuthz(query IServiceOptionDo, scope *auth.IdentityScope) IServiceOptionDo {
	q := Use(nil).ServiceOption
	if scope.ParticipantID != nil {
		return query.Where(q.ProviderID.Eq(*scope.ParticipantID))
	}
	if scope.AgentID != nil {
		// For agent scope, find provider from agent table and filter by provider
		qa := Use(nil).Agent
		providers, _ := qa.WithContext(context.Background()).Select(qa.ProviderID).Where(qa.ID.Eq(*scope.AgentID)).Find()
		if len(providers) > 0 {
			return query.Where(q.ProviderID.Eq(providers[0].ProviderID))
		}
	}
	return query
}

func applyGenServiceOptionFilters(query IServiceOptionDo, pageReq *domain.PageReq) IServiceOptionDo {
	q := Use(nil).ServiceOption
	if values, ok := pageReq.Filters["providerId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.ProviderID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}
	if values, ok := pageReq.Filters["serviceOptionTypeId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.ServiceOptionTypeID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}
	if values, ok := pageReq.Filters["enabled"]; ok && len(values) > 0 {
		for _, v := range values {
			if enabled, err := parseBool(v); err == nil {
				query = query.Where(q.Enabled.Is(enabled))
				break
			}
		}
	}
	return query
}

func applyGenServiceOptionSort(query IServiceOptionDo, pageReq *domain.PageReq) IServiceOptionDo {
	if !pageReq.Sort {
		return query
	}
	q := Use(nil).ServiceOption
	switch pageReq.SortBy {
	case "name":
		if pageReq.SortAsc {
			return query.Order(q.Name)
		}
		return query.Order(q.Name.Desc())
	case "displayOrder":
		if pageReq.SortAsc {
			return query.Order(q.DisplayOrder)
		}
		return query.Order(q.DisplayOrder.Desc())
	}
	return query
}

