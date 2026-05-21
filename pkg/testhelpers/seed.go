package testhelpers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Deterministic fixture IDs. The test Keycloak realm's `participant_id` and
// `agent_id` claims must match these UUIDs so JWTs resolve to seeded rows.
var (
	ProviderID = properties.UUID(uuid.MustParse("11111111-1111-4111-8111-111111111111"))
	ConsumerID = properties.UUID(uuid.MustParse("22222222-2222-4222-8222-222222222222"))
	AgentID    = properties.UUID(uuid.MustParse("33333333-3333-4333-8333-333333333333"))
	ServiceID  = properties.UUID(uuid.MustParse("44444444-4444-4444-8444-444444444444"))

	ConfigPoolID        = properties.UUID(uuid.MustParse("aaaaaaaa-1111-4111-8111-111111111111"))
	ConfigPoolValueID   = properties.UUID(uuid.MustParse("aaaaaaaa-2222-4222-8222-222222222222"))
	AgentInstallTokenID = properties.UUID(uuid.MustParse("aaaaaaaa-3333-4333-8333-333333333333"))
	AgentTokenID        = properties.UUID(uuid.MustParse("aaaaaaaa-4444-4444-8444-444444444444"))

	ServiceOptionTypeID = properties.UUID(uuid.MustParse("bbbbbbbb-1111-4111-8111-111111111111"))
	ServiceOptionID     = properties.UUID(uuid.MustParse("bbbbbbbb-2222-4222-8222-222222222222"))

	ServicePoolSetID   = properties.UUID(uuid.MustParse("cccccccc-1111-4111-8111-111111111111"))
	ServicePoolID      = properties.UUID(uuid.MustParse("cccccccc-2222-4222-8222-222222222222"))
	ServicePoolValueID = properties.UUID(uuid.MustParse("cccccccc-3333-4333-8333-333333333333"))

	JobID         = properties.UUID(uuid.MustParse("dddddddd-1111-4111-8111-111111111111"))
	MetricEntryID = properties.UUID(uuid.MustParse("dddddddd-2222-4222-8222-222222222222"))
	EventID       = properties.UUID(uuid.MustParse("dddddddd-3333-4333-8333-333333333333"))
)

type CoreFixtures struct {
	ServiceType       *domain.ServiceType
	MetricType        *domain.MetricType
	Provider          *domain.Participant
	Consumer          *domain.Participant
	AgentType         *domain.AgentType
	Agent             *domain.Agent
	AgentToken        *domain.Token
	AgentInstallToken *domain.AgentInstallToken
	ConfigPool        *domain.ConfigPool
	ConfigPoolValue   *domain.ConfigPoolValue
	Group             *domain.ServiceGroup
	Service           *domain.Service
	ServiceOptionType *domain.ServiceOptionType
	ServiceOption     *domain.ServiceOption
	ServicePoolSet    *domain.ServicePoolSet
	ServicePool       *domain.ServicePool
	ServicePoolValue  *domain.ServicePoolValue
	Job               *domain.Job
	MetricEntry       *domain.MetricEntry
	Event             *domain.Event
	EventSub          *domain.EventSubscription
}

// SeedCore installs the deterministic core fixture set in the caller-provided
// transaction. Expects an empty (freshly migrated) database — both the Go e2e
// harness and Playwright global-setup spin up a uniquely named DB per run, so
// no truncation/reset is performed here.
func SeedCore(tx *gorm.DB) (*CoreFixtures, error) {
	f := &CoreFixtures{}
	var err error

	if f.ServiceType, err = create(tx, &domain.ServiceType{
		Name:           "compute",
		PropertySchema: schema.Schema{},
		LifecycleSchema: domain.LifecycleSchema{
			States: []domain.LifecycleState{
				{Name: "creating"}, {Name: "created"}, {Name: "deleted"},
			},
			Actions: []domain.LifecycleAction{
				{
					Name:        "create",
					Transitions: []domain.LifecycleTransition{{From: "creating", To: "created"}},
				},
				{
					Name: "delete",
					Transitions: []domain.LifecycleTransition{
						{From: "creating", To: "deleted"},
						{From: "created", To: "deleted"},
					},
				},
			},
			InitialState:   "creating",
			TerminalStates: []string{"deleted"},
		},
	}); err != nil {
		return nil, err
	}

	if f.MetricType, err = create(tx, &domain.MetricType{
		Name:       "cpu",
		EntityType: domain.MetricEntityTypeService,
	}); err != nil {
		return nil, err
	}

	if f.Provider, err = create(tx, &domain.Participant{
		BaseEntity: domain.BaseEntity{ID: ProviderID},
		Name:       "provider1",
		Status:     domain.ParticipantEnabled,
	}); err != nil {
		return nil, err
	}

	if f.Consumer, err = create(tx, &domain.Participant{
		BaseEntity: domain.BaseEntity{ID: ConsumerID},
		Name:       "consumer1",
		Status:     domain.ParticipantEnabled,
	}); err != nil {
		return nil, err
	}

	if f.AgentType, err = create(tx, &domain.AgentType{
		Name: "vm",
		// One placeholder property so the AgentType passes the same
		// "schema must have at least one property" validator the API
		// applies on PATCH; otherwise tests that mutate this fixture
		// 400 even when not touching the schema.
		ConfigurationSchema: schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
				"placeholder": {Type: "string", Label: "Placeholder"},
			},
		},
		ConfigTemplate:    "# e2e agent config\n",
		CmdTemplate:       "curl -fsSL {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}'",
		ConfigContentType: "text/plain",
	}); err != nil {
		return nil, err
	}
	if err := tx.Model(f.AgentType).Association("ServiceTypes").Append(f.ServiceType); err != nil {
		return nil, fmt.Errorf("link AgentType→ServiceType: %w", err)
	}

	// ServicePoolSet (and the pool below it) is created before the Agent so
	// the Agent can reference it via ServicePoolSetID — the lifecycle scenario
	// rejects service creation when the agent has no pool set configured.
	if f.ServicePoolSet, err = create(tx, &domain.ServicePoolSet{
		BaseEntity: domain.BaseEntity{ID: ServicePoolSetID},
		Name:       "Mgmt network",
		ProviderID: f.Provider.ID,
	}); err != nil {
		return nil, err
	}
	if f.ServicePool, err = create(tx, &domain.ServicePool{
		BaseEntity:       domain.BaseEntity{ID: ServicePoolID},
		Name:             "Mgmt IPs",
		Type:             "mgmtIp",
		PropertyType:     "string",
		GeneratorType:    domain.PoolGeneratorList,
		ServicePoolSetID: f.ServicePoolSet.ID,
	}); err != nil {
		return nil, err
	}
	if f.ServicePoolValue, err = create(tx, &domain.ServicePoolValue{
		BaseEntity:    domain.BaseEntity{ID: ServicePoolValueID},
		Name:          "10.10.0.10",
		Value:         "10.10.0.10",
		ServicePoolID: f.ServicePool.ID,
	}); err != nil {
		return nil, err
	}

	if f.Agent, err = create(tx, &domain.Agent{
		BaseEntity:       domain.BaseEntity{ID: AgentID},
		Name:             "agent1",
		Status:           domain.AgentNew,
		ProviderID:       f.Provider.ID,
		AgentTypeID:      f.AgentType.ID,
		ServicePoolSetID: &f.ServicePoolSet.ID,
	}); err != nil {
		return nil, err
	}

	const agentTokenPlain = "e2e-agent-token"
	if f.AgentToken, err = create(tx, &domain.Token{
		BaseEntity:  domain.BaseEntity{ID: AgentTokenID},
		Name:        "agent1 bootstrap",
		Role:        auth.RoleAgent,
		HashedValue: domain.HashTokenValue(agentTokenPlain),
		ExpireAt:    time.Now().Add(365 * 24 * time.Hour),
		AgentID:     &f.Agent.ID,
	}); err != nil {
		return nil, err
	}

	const installPlain = "e2e-install-token"
	installSum := sha256.Sum256([]byte(installPlain))
	if f.AgentInstallToken, err = create(tx, &domain.AgentInstallToken{
		BaseEntity:  domain.BaseEntity{ID: AgentInstallTokenID},
		AgentID:     f.Agent.ID,
		TokenHashed: hex.EncodeToString(installSum[:]),
		ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
	}); err != nil {
		return nil, err
	}

	if f.ConfigPool, err = create(tx, &domain.ConfigPool{
		BaseEntity:    domain.BaseEntity{ID: ConfigPoolID},
		Name:          "Internal IPs",
		Type:          "internalIp",
		PropertyType:  "string",
		GeneratorType: domain.PoolGeneratorList,
	}); err != nil {
		return nil, err
	}
	if f.ConfigPoolValue, err = create(tx, &domain.ConfigPoolValue{
		BaseEntity:   domain.BaseEntity{ID: ConfigPoolValueID},
		Name:         "10.0.0.10",
		Value:        "10.0.0.10",
		ConfigPoolID: f.ConfigPool.ID,
	}); err != nil {
		return nil, err
	}

	if f.Group, err = create(tx, &domain.ServiceGroup{
		Name:       "default-group",
		ConsumerID: f.Consumer.ID,
	}); err != nil {
		return nil, err
	}

	if f.Service, err = create(tx, &domain.Service{
		BaseEntity:    domain.BaseEntity{ID: ServiceID},
		Name:          "service1",
		Status:        "creating",
		ProviderID:    f.Provider.ID,
		ConsumerID:    f.Consumer.ID,
		GroupID:       f.Group.ID,
		AgentID:       f.Agent.ID,
		ServiceTypeID: f.ServiceType.ID,
	}); err != nil {
		return nil, err
	}

	if f.ServiceOptionType, err = create(tx, &domain.ServiceOptionType{
		BaseEntity:  domain.BaseEntity{ID: ServiceOptionTypeID},
		Name:        "CPU",
		Type:        "cpu",
		Description: "CPU sizing",
	}); err != nil {
		return nil, err
	}
	enabled := true
	if f.ServiceOption, err = create(tx, &domain.ServiceOption{
		BaseEntity:          domain.BaseEntity{ID: ServiceOptionID},
		ProviderID:          f.Provider.ID,
		ServiceOptionTypeID: f.ServiceOptionType.ID,
		Name:                "2 vCPU",
		Value:               "2",
		Enabled:             &enabled,
		DisplayOrder:        0,
	}); err != nil {
		return nil, err
	}

	if f.Job, err = create(tx, &domain.Job{
		BaseEntity: domain.BaseEntity{ID: JobID},
		Action:     "service.create",
		Priority:   1,
		Status:     domain.JobPending,
		AgentID:    f.Agent.ID,
		ServiceID:  f.Service.ID,
		ProviderID: f.Provider.ID,
		ConsumerID: f.Consumer.ID,
	}); err != nil {
		return nil, err
	}

	if f.MetricEntry, err = create(tx, &domain.MetricEntry{
		ID:         MetricEntryID,
		ResourceID: "vm-0",
		Value:      0.42,
		TypeID:     f.MetricType.ID,
		AgentID:    f.Agent.ID,
		ServiceID:  f.Service.ID,
		ProviderID: f.Provider.ID,
		ConsumerID: f.Consumer.ID,
	}); err != nil {
		return nil, err
	}

	entityID := f.Service.ID
	if f.Event, err = create(tx, &domain.Event{
		BaseEntity:    domain.BaseEntity{ID: EventID},
		InitiatorType: domain.InitiatorTypeSystem,
		InitiatorID:   "e2e-seed",
		Type:          domain.EventTypeServiceCreated,
		Payload:       properties.JSON{},
		EntityID:      &entityID,
		ProviderID:    &f.Provider.ID,
		ConsumerID:    &f.Consumer.ID,
		AgentID:       &f.Agent.ID,
	}); err != nil {
		return nil, err
	}

	if f.EventSub, err = create(tx, &domain.EventSubscription{
		SubscriberID: "e2e-subscriber",
		IsActive:     true,
	}); err != nil {
		return nil, err
	}

	return f, nil
}

func create[T any](tx *gorm.DB, m *T) (*T, error) {
	if err := tx.Create(m).Error; err != nil {
		return nil, fmt.Errorf("create %T: %w", m, err)
	}
	return m, nil
}
