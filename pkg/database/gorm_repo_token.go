package database

import (
	"context"
	"log/slog"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormTokenRepository struct {
	*GormRepository[domain.Token]
}

var applyTokenFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"name": StringContainsInsensitiveFilterFieldApplier("name"),
	"role": StringInFilterFieldApplier("role"),
	"participantId": ParserInFilterFieldApplier("participant_id", properties.ParseUUID),
	"agentId": ParserInFilterFieldApplier("agent_id", properties.ParseUUID),
})

var applyTokenSort = MapSortApplier(map[string]string{
	"name":      "name",
	"expireAt":  "expire_at",
	"createdAt": "created_at",
})

// NewTokenRepository creates a new instance of TokenRepository
func NewTokenRepository(db *gorm.DB) *GormTokenRepository {
	repo := &GormTokenRepository{
		GormRepository: NewGormRepository[domain.Token](
			db,
			applyTokenFilter,
			applyTokenSort,
			participantAuthzFilterApplier,
			[]string{}, // No preload paths needed for finding by ID
			[]string{}, // No preload paths needed for list
		),
	}
	return repo
}

// FindByHashedValue finds a token by its hashed value
func (r *GormTokenRepository) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	var token domain.Token
	err := r.db.WithContext(ctx).
		Model(&domain.Token{}).
		Where("hashed_value = ?", hashedValue).
		First(&token).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	slog.Debug("token retrived", slog.Any("id", token.ID), slog.Any("role", token.Role), slog.String("name", token.Name))
	return &token, nil
}

// DeleteByAgentID removes all tokens associated with an agent ID
func (r *GormTokenRepository) DeleteByAgentID(ctx context.Context, agentID properties.UUID) error {
	// Delete all tokens with the given agent ID
	result := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&domain.Token{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// DeleteByParticipantID removes all tokens associated with a participant ID
func (r *GormTokenRepository) DeleteByParticipantID(ctx context.Context, participantID properties.UUID) error {
	// Delete all tokens with the given participant ID
	result := r.db.WithContext(ctx).Where("participant_id = ?", participantID).Delete(&domain.Token{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// AuthScope returns the auth scope for the token
func (r *GormTokenRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "participant_id", "null", "agent_id", "null")
}
