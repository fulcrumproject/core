// Participant repository implementation using GORM Gen
// Provides type-safe database operations for Participant entities
package database

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenParticipantRepository struct {
	q *Query
}

func NewGenParticipantRepository(db *gorm.DB) *GenParticipantRepository {
	return &GenParticipantRepository{
		q: Use(db),
	}
}

// Create inserts a new participant
func (r *GenParticipantRepository) Create(ctx context.Context, participant *domain.Participant) error {
	return r.q.Participant.WithContext(ctx).Create(participant)
}

// Save updates an existing participant
func (r *GenParticipantRepository) Save(ctx context.Context, participant *domain.Participant) error {
	result, err := r.q.Participant.WithContext(ctx).
		Where(r.q.Participant.ID.Eq(participant.ID)).
		Updates(participant)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

// Delete removes a participant by ID
func (r *GenParticipantRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.Participant.WithContext(ctx).
		Where(r.q.Participant.ID.Eq(id)).
		Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

// Get retrieves a participant by ID
func (r *GenParticipantRepository) Get(ctx context.Context, id properties.UUID) (*domain.Participant, error) {
	participant, err := r.q.Participant.WithContext(ctx).
		Where(r.q.Participant.ID.Eq(id)).
		First()

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return participant, nil
}

// Exists checks if a participant exists
func (r *GenParticipantRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.Participant.WithContext(ctx).
		Where(r.q.Participant.ID.Eq(id)).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count returns total count of all participants
func (r *GenParticipantRepository) Count(ctx context.Context) (int64, error) {
	return r.q.Participant.WithContext(ctx).Count()
}

// List returns paginated participants with authorization, filters, and sorting
func (r *GenParticipantRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.Participant], error) {
	query := r.q.Participant.WithContext(ctx)
	query = applyGenParticipantAuthz(query, scope)

	result, err := PaginateQuery[domain.Participant, IParticipantDo](
		ctx,
		query,
		pageReq,
		applyGenParticipantFilters,
		applyGenParticipantSort,
	)
	if err != nil {
		return nil, err
	}

	// Convert []*Participant to []Participant to match interface
	items := make([]domain.Participant, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}

	return &domain.PageRes[domain.Participant]{
		Items:       items,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	}, nil
}

// AuthScope returns the authorization scope for a participant
func (r *GenParticipantRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	participant, err := r.q.Participant.WithContext(ctx).
		Select(r.q.Participant.ID).
		Where(r.q.Participant.ID.Eq(id)).
		First()

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}

	return &auth.DefaultObjectScope{
		ParticipantID: &participant.ID,
	}, nil
}

// applyGenParticipantAuthz applies authorization filters
func applyGenParticipantAuthz(query IParticipantDo, scope *auth.IdentityScope) IParticipantDo {
	q := Use(nil).Participant

	if scope.ParticipantID != nil {
		return query.Where(q.ID.Eq(*scope.ParticipantID))
	}
	return query
}

// applyGenParticipantFilters applies request filters
func applyGenParticipantFilters(query IParticipantDo, pageReq *domain.PageReq) IParticipantDo {
	q := Use(nil).Participant

	if values, ok := pageReq.Filters["name"]; ok && len(values) > 0 {
		query = query.Where(q.Name.In(values...))
	}

	if values, ok := pageReq.Filters["status"]; ok && len(values) > 0 {
		statuses := make([]string, 0, len(values))
		for _, v := range values {
			if status, err := domain.ParseParticipantStatus(v); err == nil {
				statuses = append(statuses, string(status))
			}
		}
		if len(statuses) > 0 {
			query = query.Where(q.Status.In(statuses...))
		}
	}

	return query
}

// applyGenParticipantSort applies sorting
func applyGenParticipantSort(query IParticipantDo, pageReq *domain.PageReq) IParticipantDo {
	if !pageReq.Sort {
		return query
	}

	q := Use(nil).Participant

	switch pageReq.SortBy {
	case "name":
		if pageReq.SortAsc {
			query = query.Order(q.Name)
		} else {
			query = query.Order(q.Name.Desc())
		}
	}

	return query
}
