package domain

// AgentType represents a type of service manager agent
type AgentType struct {
	BaseEntity
	Name         string        `json:"name" gorm:"not null;unique"`
	ServiceTypes []ServiceType `json:"-" gorm:"many2many:agent_type_service_types;"`
}

// TableName returns the table name for the agent type
func (AgentType) TableName() string {
	return "agent_types"
}

// AgentTypeRepository defines the interface for the AgentType repository
type AgentTypeRepository interface {
	AgentTypeQuerier
	BaseEntityRepository[AgentType]
}

// AgentTypeQuerier defines the interface for the AgentType read-only queries
type AgentTypeQuerier interface {
	BaseEntityQuerier[AgentType]
}
