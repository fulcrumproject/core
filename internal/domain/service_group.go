package domain

// ServiceGroup represents a group of related services
type ServiceGroup struct {
	BaseEntity
	Name     string    `gorm:"not null"`
	Services []Service `gorm:"foreignKey:GroupID"`
}

// TableName returns the table name for the service
func (ServiceGroup) TableName() string {
	return "service_groups"
}
