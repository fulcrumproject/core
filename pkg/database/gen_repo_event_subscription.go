// EventSubscription repository implementation using GORM Gen
// Provides type-safe database operations for EventSubscription entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenEventSubscriptionRepository struct {
	q *Query
}

func NewGenEventSubscriptionRepository(db *gorm.DB) *GenEventSubscriptionRepository {
	return &GenEventSubscriptionRepository{q: Use(db)}
}

func (r *GenEventSubscriptionRepository) Create(ctx context.Context, entity *domain.EventSubscription) error {
	return r.q.EventSubscription.WithContext(ctx).Create(entity)
}

func (r *GenEventSubscriptionRepository) Save(ctx context.Context, entity *domain.EventSubscription) error {
	result, err := r.q.EventSubscription.WithContext(ctx).Where(r.q.EventSubscription.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenEventSubscriptionRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.EventSubscription.WithContext(ctx).Where(r.q.EventSubscription.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenEventSubscriptionRepository) Get(ctx context.Context, id properties.UUID) (*domain.EventSubscription, error) {
	entity, err := r.q.EventSubscription.WithContext(ctx).Where(r.q.EventSubscription.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenEventSubscriptionRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.EventSubscription.WithContext(ctx).Where(r.q.EventSubscription.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenEventSubscriptionRepository) Count(ctx context.Context) (int64, error) {
	return r.q.EventSubscription.WithContext(ctx).Count()
}

func (r *GenEventSubscriptionRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.EventSubscription], error) {
	query := r.q.EventSubscription.WithContext(ctx)

	result, err := PaginateQuery(ctx, query, pageReq,
		applyGenEventSubscriptionFilters,
		applyGenEventSubscriptionSort,
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.EventSubscription, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.EventSubscription]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenEventSubscriptionRepository) FindBySubscriberID(ctx context.Context, subscriberID string) (*domain.EventSubscription, error) {
	entity, err := r.q.EventSubscription.WithContext(ctx).
		Where(r.q.EventSubscription.SubscriberID.Eq(subscriberID)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundErrorf("event subscription with subscriber_id %s", subscriberID)
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenEventSubscriptionRepository) ExistsBySubscriberID(ctx context.Context, subscriberID string) (bool, error) {
	count, err := r.q.EventSubscription.WithContext(ctx).
		Where(r.q.EventSubscription.SubscriberID.Eq(subscriberID)).
		Count()
	return count > 0, err
}

func (r *GenEventSubscriptionRepository) DeleteBySubscriberID(ctx context.Context, subscriberID string) error {
	result, err := r.q.EventSubscription.WithContext(ctx).
		Where(r.q.EventSubscription.SubscriberID.Eq(subscriberID)).
		Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NewNotFoundErrorf("event subscription with subscriber_id %s", subscriberID)
	}
	return nil
}

func (r *GenEventSubscriptionRepository) ListExpiredLeases(ctx context.Context) ([]*domain.EventSubscription, error) {
	var subscriptions []*domain.EventSubscription
	err := r.q.EventSubscription.WithContext(ctx).UnderlyingDB().
		Where("lease_expires_at IS NOT NULL AND lease_expires_at < NOW()").
		Find(&subscriptions).Error
	return subscriptions, err
}

func (r *GenEventSubscriptionRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	exists, err := r.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return &auth.AllwaysMatchObjectScope{}, nil
}

func applyGenEventSubscriptionFilters(query IEventSubscriptionDo, pageReq *domain.PageReq) IEventSubscriptionDo {
	q := Use(nil).EventSubscription
	if values, ok := pageReq.Filters["subscriber_id"]; ok && len(values) > 0 {
		query = query.Where(q.SubscriberID.In(values...))
	}
	if values, ok := pageReq.Filters["is_active"]; ok && len(values) > 0 {
		for _, v := range values {
			if active, err := parseBool(v); err == nil {
				query = query.Where(q.IsActive.Is(active))
				break
			}
		}
	}
	return query
}

func applyGenEventSubscriptionSort(query IEventSubscriptionDo, pageReq *domain.PageReq) IEventSubscriptionDo {
	if !pageReq.Sort {
		return query
	}
	q := Use(nil).EventSubscription
	switch pageReq.SortBy {
	case "subscriber_id":
		if pageReq.SortAsc {
			return query.Order(q.SubscriberID)
		}
		return query.Order(q.SubscriberID.Desc())
	case "last_event_sequence_processed":
		if pageReq.SortAsc {
			return query.Order(q.LastEventSequenceProcessed)
		}
		return query.Order(q.LastEventSequenceProcessed.Desc())
	case "lease_expires_at":
		if pageReq.SortAsc {
			return query.Order(q.LeaseExpiresAt)
		}
		return query.Order(q.LeaseExpiresAt.Desc())
	case "createdAt":
		if pageReq.SortAsc {
			return query.Order(q.CreatedAt)
		}
		return query.Order(q.CreatedAt.Desc())
	case "updatedAt":
		if pageReq.SortAsc {
			return query.Order(q.UpdatedAt)
		}
		return query.Order(q.UpdatedAt.Desc())
	}
	return query
}

