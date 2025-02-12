package domain

import "gorm.io/gorm"

// ServiceState represents the possible states of a service
type ServiceState string

const (
	ServiceNew      ServiceState = "New"
	ServiceCreating ServiceState = "Creating"
	ServiceCreated  ServiceState = "Created"
	ServiceUpdating ServiceState = "Updating"
	ServiceUpdated  ServiceState = "Updated"
	ServiceDeleting ServiceState = "Deleting"
	ServiceDeleted  ServiceState = "Deleted"
	ServiceError    ServiceState = "Error"
)

// Service represents a service instance managed by an agent
type Service struct {
	BaseEntity
	Name           string         `gorm:"not null" json:"name"`
	State          ServiceState   `gorm:"not null" json:"state"`
	Attributes     Attributes     `gorm:"-" json:"attributes"`
	GormAttributes GormAttributes `gorm:"column:attributes;type:jsonb" json:"-"`
	Resources      JSON           `gorm:"-" json:"resources"`
	GormResources  GormJSON       `gorm:"column:resources;type:jsonb" json:"-"`
	AgentID        UUID           `gorm:"not null" json:"agentId"`
	ServiceTypeID  UUID           `gorm:"not null" json:"serviceTypeId"`
	GroupID        UUID           `json:"groupId"`

	// Relationships
	Agent       *Agent        `gorm:"foreignKey:AgentID" json:"-"`
	ServiceType *ServiceType  `gorm:"foreignKey:ServiceTypeID" json:"-"`
	Group       *ServiceGroup `gorm:"foreignKey:GroupID" json:"-"`
}

// BeforeCreate ensures the service is in a valid state before creation
func (s *Service) BeforeCreate(tx *gorm.DB) error {
	if err := s.BaseEntity.BeforeCreate(tx); err != nil {
		return err
	}

	if s.State == "" {
		s.State = ServiceNew
	}

	if s.Attributes != nil {
		gormAttrs, err := s.Attributes.ToGormAttributes()
		if err != nil {
			return err
		}
		s.GormAttributes = gormAttrs
	}

	if s.Resources != nil {
		gormJSON, err := s.Resources.ToGormJSON()
		if err != nil {
			return err
		}
		s.GormResources = gormJSON
	}

	return nil
}

// AfterFind populates the Attributes and Resources fields from their GORM counterparts
func (s *Service) AfterFind(tx *gorm.DB) error {
	if s.GormAttributes != nil {
		attrs, err := s.GormAttributes.ToAttributes()
		if err != nil {
			return err
		}
		s.Attributes = attrs
	}

	if s.GormResources != nil {
		resources, err := s.GormResources.ToJSON()
		if err != nil {
			return err
		}
		s.Resources = resources
	}

	return nil
}

// BeforeSave ensures the GORM fields are updated before saving
func (s *Service) BeforeSave(tx *gorm.DB) error {
	if s.Attributes != nil {
		gormAttrs, err := s.Attributes.ToGormAttributes()
		if err != nil {
			return err
		}
		s.GormAttributes = gormAttrs
	}

	if s.Resources != nil {
		gormJSON, err := s.Resources.ToGormJSON()
		if err != nil {
			return err
		}
		s.GormResources = gormJSON
	}

	return nil
}
