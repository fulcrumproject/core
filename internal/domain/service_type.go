package domain

// ServiceType represents a type of service that can be provided
type ServiceType struct {
	BaseEntity
	Name                string      `gorm:"not null;unique"`
	ResourceDefinitions JSON        `gorm:"type:jsonb"`
	AgentTypes          []AgentType `gorm:"many2many:agent_type_service_types"`
}

// TableName returns the table name for the service type
func (ServiceType) TableName() string {
	return "service_types"
}
