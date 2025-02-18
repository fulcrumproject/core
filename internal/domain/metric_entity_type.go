package domain

// MetricEntityType represents the possible types of entities that can be measured
type MetricEntityType string

const (
	MetricEntityTypeAgent    MetricEntityType = "Agent"
	MetricEntityTypeService  MetricEntityType = "Service"
	MetricEntityTypeResource MetricEntityType = "Resource"
)
