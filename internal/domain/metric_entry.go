package domain

// MetricEntry represents a metric measurement for a specific resource
type MetricEntry struct {
	BaseEntity
	AgentID    UUID    `gorm:"not null"`
	ServiceID  UUID    `gorm:"not null"`
	ResourceID string  `gorm:"not null"`
	Value      float64 `gorm:"not null"`
	TypeID     UUID    `gorm:"not null"`

	// Relationships
	Agent   *Agent      `gorm:"foreignKey:AgentID"`
	Service *Service    `gorm:"foreignKey:ServiceID"`
	Type    *MetricType `gorm:"foreignKey:TypeID"`
}

// TableName returns the table name for the metric entry
func (MetricEntry) TableName() string {
	return "metric_entries"
}
