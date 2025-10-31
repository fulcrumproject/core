// Token repository implementation using GORM Gen
// Provides type-safe database operations for Token entities
package database

import (
	"context"
	"log/slog"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"
)

type GenTokenRepository struct {
	q *Query
}

func NewGenTokenRepository(db *gorm.DB) *GenTokenRepository {
	return &GenTokenRepository{
		q: Use(db),
	}
}

// Create inserts a new token
func (r *GenTokenRepository) Create(ctx context.Context, token *domain.Token) error {
	return r.q.Token.WithContext(ctx).Create(token)
}

// Save updates an existing token
func (r *GenTokenRepository) Save(ctx context.Context, token *domain.Token) error {
	result, err := r.q.Token.WithContext(ctx).
		Where(r.q.Token.ID.Eq(token.ID)).
		Updates(token)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

// Delete removes a token by ID
func (r *GenTokenRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.Token.WithContext(ctx).
		Where(r.q.Token.ID.Eq(id)).
		Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

// Get retrieves a token by ID
func (r *GenTokenRepository) Get(ctx context.Context, id properties.UUID) (*domain.Token, error) {
	token, err := r.q.Token.WithContext(ctx).
		Where(r.q.Token.ID.Eq(id)).
		First()
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	
	return token, nil
}

// Exists checks if a token exists
func (r *GenTokenRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.Token.WithContext(ctx).
		Where(r.q.Token.ID.Eq(id)).
		Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Count returns total count of all tokens
func (r *GenTokenRepository) Count(ctx context.Context) (int64, error) {
	return r.q.Token.WithContext(ctx).Count()
}

// List returns paginated tokens with authorization, filters, and sorting
func (r *GenTokenRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.Token], error) {
	query := r.q.Token.WithContext(ctx)
	query = applyGenTokenAuthz(query, scope)

	result, err := PaginateQuery(
		ctx,
		query,
		pageReq,
		applyGenTokenFilters,
		applyGenTokenSort,
	)
	if err != nil {
		return nil, err
	}

	// Convert []*Token to []Token to match interface
	items := make([]domain.Token, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}

	return &domain.PageRes[domain.Token]{
		Items:       items,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	}, nil
}

// FindByHashedValue finds a token by its hashed value
func (r *GenTokenRepository) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	token, err := r.q.Token.WithContext(ctx).
		Where(r.q.Token.HashedValue.Eq(hashedValue)).
		First()
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	
	slog.Debug("token retrieved", slog.Any("id", token.ID), slog.Any("role", token.Role), slog.String("name", token.Name))
	return token, nil
}

// DeleteByAgentID removes all tokens associated with an agent ID
func (r *GenTokenRepository) DeleteByAgentID(ctx context.Context, agentID properties.UUID) error {
	_, err := r.q.Token.WithContext(ctx).
		Where(r.q.Token.AgentID.Eq(agentID)).
		Delete()
	return err
}

// DeleteByParticipantID removes all tokens associated with a participant ID
func (r *GenTokenRepository) DeleteByParticipantID(ctx context.Context, participantID properties.UUID) error {
	_, err := r.q.Token.WithContext(ctx).
		Where(r.q.Token.ParticipantID.Eq(participantID)).
		Delete()
	return err
}

// AuthScope returns the authorization scope for a token
func (r *GenTokenRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	token, err := r.q.Token.WithContext(ctx).
		Select(r.q.Token.ParticipantID, r.q.Token.AgentID).
		Where(r.q.Token.ID.Eq(id)).
		First()
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	
	return &auth.DefaultObjectScope{
		ParticipantID: token.ParticipantID,
		AgentID:       token.AgentID,
	}, nil
}

// applyGenTokenAuthz applies authorization filters
func applyGenTokenAuthz(query ITokenDo, scope *auth.IdentityScope) ITokenDo {
	q := Use(nil).Token
	
	if scope.ParticipantID != nil {
		return query.Where(q.ParticipantID.Eq(*scope.ParticipantID))
	}
	return query
}

// applyGenTokenFilters applies request filters
func applyGenTokenFilters(query ITokenDo, pageReq *domain.PageReq) ITokenDo {
	q := Use(nil).Token

	if values, ok := pageReq.Filters["name"]; ok && len(values) > 0 {
		query = query.Where(q.Name.In(values...))
	}

	if values, ok := pageReq.Filters["role"]; ok && len(values) > 0 {
		query = query.Where(q.Role.In(values...))
	}

	return query
}

// applyGenTokenSort applies sorting
func applyGenTokenSort(query ITokenDo, pageReq *domain.PageReq) ITokenDo {
	if !pageReq.Sort {
		return query
	}

	q := Use(nil).Token

	switch pageReq.SortBy {
	case "name":
		if pageReq.SortAsc {
			query = query.Order(q.Name)
		} else {
			query = query.Order(q.Name.Desc())
		}
	case "expireAt":
		if pageReq.SortAsc {
			query = query.Order(q.ExpireAt)
		} else {
			query = query.Order(q.ExpireAt.Desc())
		}
	case "createdAt":
		if pageReq.SortAsc {
			query = query.Order(q.CreatedAt)
		} else {
			query = query.Order(q.CreatedAt.Desc())
		}
	}

	return query
}

