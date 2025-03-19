package domain

import (
	"context"
)

// MetricEntry represents a metric measurement for a specific resource
type MetricEntry struct {
	BaseEntity

	ResourceID string  `gorm:"not null"`
	Value      float64 `gorm:"not null"`

	// Relationships
	TypeID    UUID        `gorm:"not null"`
	Type      *MetricType `gorm:"foreignKey:TypeID"`
	AgentID   UUID        `gorm:"not null"`
	Agent     *Agent      `gorm:"foreignKey:AgentID"`
	ServiceID UUID        `gorm:"not null"`
	Service   *Service    `gorm:"foreignKey:ServiceID"`
}

// TableName returns the table name for the metric entry
func (MetricEntry) TableName() string {
	return "metric_entries"
}

// Validate ensures all MetricEntry fields are valid
func (p *MetricEntry) Validate() error {
	// TODO
	return nil
}

// MetricEntryCommander defines the interface for metric entry command operations
type MetricEntryCommander interface {
	// Create creates a new metric entry
	Create(ctx context.Context, typeName string, agentID UUID, serviceID UUID, resourceID string, value float64) (*MetricEntry, error)

	// CreateWithExternalID creates a new metric entry using service's external ID
	CreateWithExternalID(ctx context.Context, typeName string, agentID UUID, externalID string, resourceID string, value float64) (*MetricEntry, error)
}

// metricEntryCommander is the concrete implementation of MetricEntryCommander
type metricEntryCommander struct {
	store Store
}

// NewMetricEntryCommander creates a new MetricEntryService
func NewMetricEntryCommander(
	store Store,
) *metricEntryCommander {
	return &metricEntryCommander{
		store: store,
	}
}

func (s *metricEntryCommander) CreateWithExternalID(
	ctx context.Context,
	typeName string,
	agentID UUID,
	externalID string,
	resourceID string,
	value float64,
) (*MetricEntry, error) {
	// Get the agent to retrieve its providerID
	agent, err := s.store.AgentRepo().FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Look up the service by external ID to get the service group and broker ID
	service, err := s.store.ServiceRepo().FindByExternalID(ctx, agentID, externalID)
	if err != nil {
		return nil, err
	}

	// Get the service group to get the broker ID
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, service.GroupID)
	if err != nil {
		return nil, err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{AgentID: &agentID, ProviderID: &agent.ProviderID, BrokerID: &sg.BrokerID}); err != nil {
		return nil, err
	}

	// Look up the service by external ID
	return s.Create(ctx, typeName, agentID, service.ID, resourceID, value)
}

func (s *metricEntryCommander) Create(
	ctx context.Context,
	typeName string,
	agentID UUID,
	serviceID UUID,
	resourceID string,
	value float64,
) (*MetricEntry, error) {
	// Get the agent to retrieve its providerID
	agent, err := s.store.AgentRepo().FindByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Look up the service by external ID
	service, err := s.store.ServiceRepo().FindByID(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	// Get the service group to get the broker ID
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, service.GroupID)
	if err != nil {
		return nil, err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{AgentID: &agentID, ProviderID: &agent.ProviderID, BrokerID: &sg.BrokerID}); err != nil {
		return nil, err
	}

	// Look up the service by external ID
	// Look up the service type by name
	metricType, err := s.store.MetricTypeRepo().FindByName(ctx, typeName)
	if err != nil {
		return nil, err
	}
	// TODO check id's exist with cache
	metricEntry := &MetricEntry{
		AgentID:    agentID,
		ServiceID:  service.ID,
		ResourceID: resourceID,
		Value:      value,
		TypeID:     metricType.ID,
	}
	if err := metricEntry.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.MetricEntryRepo().Create(ctx, metricEntry); err != nil {
		return nil, err
	}
	return metricEntry, nil
}

type MetricEntryRepository interface {
	MetricEntryQuerier

	// Create creates a new metric entry
	Create(ctx context.Context, entity *MetricEntry) error
}

type MetricEntryQuerier interface {
	// List retrieves a list of metric entries based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[MetricEntry], error)

	// CountByMetricType counts the number of entries for a specific metric type
	CountByMetricType(ctx context.Context, typeID UUID) (int64, error)
}
