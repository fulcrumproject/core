package domain

import "context"

// AuditEntryRepository defines the interface for audit entry persistence operations
type AuditEntryRepository interface {
	// Create stores a new audit entry
	Create(ctx context.Context, entry *AuditEntry) error

	// List retrieves a list of audit entries based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[AuditEntry], error)
}
