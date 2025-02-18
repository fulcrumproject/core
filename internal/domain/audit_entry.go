package domain

// AuditEntry represents an audit log entry
type AuditEntry struct {
	BaseEntity
	AuthorityType string `gorm:"not null"`
	AuthorityID   string `gorm:"not null"`
	Type          string `gorm:"not null"`
	Properties    JSON   `gorm:"column:properties;type:jsonb"`
}

// TableName returns the table name for the audit entry
func (AuditEntry) TableName() string {
	return "audit_entries"
}
