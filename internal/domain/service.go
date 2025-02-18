package domain

// Service represents a service instance managed by an agent
type Service struct {
	BaseEntity
	Name          string                 `gorm:"not null"`
	State         ServiceState           `gorm:"not null"`
	Attributes    Attributes             `gorm:"column:attributes;type:jsonb"`
	Resources     map[string]interface{} `gorm:"column:resources;type:jsonb"`
	AgentID       UUID                   `gorm:"not null"`
	ServiceTypeID UUID                   `gorm:"not null"`
	GroupID       UUID                   `json:"groupId"`

	// Relationships
	Agent       *Agent        `gorm:"foreignKey:AgentID"`
	ServiceType *ServiceType  `gorm:"foreignKey:ServiceTypeID"`
	Group       *ServiceGroup `gorm:"foreignKey:GroupID"`
}

// TableName returns the table name for the service
func (Service) TableName() string {
	return "services"
}
