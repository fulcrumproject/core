package domain

// MetricType represents a type of metric that can be collected
type MetricType struct {
	BaseEntity
	Name       string           `gorm:"not null"`
	EntityType MetricEntityType `gorm:"not null"`
}

// TableName returns the table name for the metric type
func (MetricType) TableName() string {
	return "metric_types"
}
