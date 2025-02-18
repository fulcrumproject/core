package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type auditEntryRepository struct {
	db *gorm.DB
}

// NewAuditEntryRepository creates a new instance of AuditEntryRepository
func NewAuditEntryRepository(db *gorm.DB) domain.AuditEntryRepository {
	return &auditEntryRepository{db: db}
}

func (r *auditEntryRepository) Create(ctx context.Context, auditEntry *domain.AuditEntry) error {
	result := r.db.WithContext(ctx).Create(auditEntry)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

var auditEntryFilterConfigs = map[string]FilterConfig{
	"authorityType": {Query: "authority_type"},
	"authorityId":   {Query: "authority_id"},
	"type":          {Query: "type"},
}

func (r *auditEntryRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.AuditEntry], error) {
	var auditEntries []domain.AuditEntry
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&domain.AuditEntry{})

	query, totalItems, err := applyFindAndCount(query, filter, auditEntryFilterConfigs, sorting, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&auditEntries).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(auditEntries, totalItems, pagination), nil
}
