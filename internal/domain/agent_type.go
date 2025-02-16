package domain

// AgentType represents a type of service manager agent
type AgentType struct {
	BaseEntity
	Name         string        `gorm:"not null;unique"`
	ServiceTypes []ServiceType `gorm:"many2many:agent_type_service_types;"`
}

// TableName returns the table name for the agent type
func (AgentType) TableName() string {
	return "agent_types"
}
