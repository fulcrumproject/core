package domain

// ServiceGroup represents a group of related services
type ServiceGroup struct {
	BaseEntity
	Name     string    `gorm:"not null" json:"name"`
	Services []Service `gorm:"foreignKey:GroupID" json:"services"`
}
