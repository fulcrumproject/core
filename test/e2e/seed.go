package e2e

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Deterministic Participant IDs. The Keycloak realm's `participant_id` claim
// for participant1 / consumer1 must match these UUIDs for role-scoped tokens
// to resolve to the seeded rows.
var (
	providerID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	consumerID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
)

type Fixtures struct {
	ServiceType  *domain.ServiceType
	OptionType   *domain.ServiceOptionType
	MetricType   *domain.MetricType
	Provider     *domain.Participant
	Consumer     *domain.Participant
	AgentType    *domain.AgentType
	AgentPool    *domain.AgentPool
	PoolSet      *domain.ServicePoolSet
	ServicePool  *domain.ServicePool
	Agent        *domain.Agent
	InstallToken *domain.AgentInstallToken
	PoolValue    *domain.AgentPoolValue
	Option       *domain.ServiceOption
	Group        *domain.ServiceGroup
	Service      *domain.Service
	PoolValueSvc *domain.ServicePoolValue
	EventSub     *domain.EventSubscription
}

func mustSeed(t *testing.T, db *gorm.DB) *Fixtures {
	t.Helper()
	f := &Fixtures{}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		f.ServiceType = mustCreate(t, tx, &domain.ServiceType{
			Name:            "compute",
			PropertySchema:  schema.Schema{},
			LifecycleSchema: domain.LifecycleSchema{InitialState: "creating"},
		})
		f.OptionType = mustCreate(t, tx, &domain.ServiceOptionType{
			Name: "size",
			Type: "size-test",
		})
		f.MetricType = mustCreate(t, tx, &domain.MetricType{
			Name:       "cpu",
			EntityType: domain.MetricEntityTypeService,
		})
		f.Provider = mustCreate(t, tx, &domain.Participant{
			BaseEntity: domain.BaseEntity{ID: providerID},
			Name:       "provider1",
			Status:     domain.ParticipantEnabled,
		})
		f.Consumer = mustCreate(t, tx, &domain.Participant{
			BaseEntity: domain.BaseEntity{ID: consumerID},
			Name:       "consumer1",
			Status:     domain.ParticipantEnabled,
		})
		f.AgentType = mustCreate(t, tx, &domain.AgentType{
			Name:                "vm",
			ConfigurationSchema: schema.Schema{},
		})
		f.AgentPool = mustCreate(t, tx, &domain.AgentPool{
			Name:          "default",
			Type:          "agent-pool-test",
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
		})
		f.PoolSet = mustCreate(t, tx, &domain.ServicePoolSet{
			Name:       "default-set",
			ProviderID: f.Provider.ID,
		})
		f.ServicePool = mustCreate(t, tx, &domain.ServicePool{
			Name:             "default-pool",
			Type:             "service-pool-test",
			PropertyType:     "string",
			GeneratorType:    domain.PoolGeneratorList,
			ServicePoolSetID: f.PoolSet.ID,
		})
		f.Agent = mustCreate(t, tx, &domain.Agent{
			Name:             "agent1",
			Status:           domain.AgentNew,
			ProviderID:       f.Provider.ID,
			AgentTypeID:      f.AgentType.ID,
			ServicePoolSetID: &f.PoolSet.ID,
		})
		f.InstallToken = mustCreate(t, tx, &domain.AgentInstallToken{
			AgentID:     f.Agent.ID,
			TokenHashed: "seed-hash-" + properties.NewUUID().String(),
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		})
		f.PoolValue = mustCreate(t, tx, &domain.AgentPoolValue{
			Name:        "pool-value-1",
			Value:       "v1",
			AgentPoolID: f.AgentPool.ID,
			AgentID:     &f.Agent.ID,
		})
		enabled := true
		f.Option = mustCreate(t, tx, &domain.ServiceOption{
			ProviderID:          f.Provider.ID,
			ServiceOptionTypeID: f.OptionType.ID,
			Name:                "small",
			Value:               "small",
			Enabled:             &enabled,
		})
		f.Group = mustCreate(t, tx, &domain.ServiceGroup{
			Name:       "default-group",
			ConsumerID: f.Consumer.ID,
		})
		f.Service = mustCreate(t, tx, &domain.Service{
			Name:          "service1",
			Status:        "creating",
			ProviderID:    f.Provider.ID,
			ConsumerID:    f.Consumer.ID,
			GroupID:       f.Group.ID,
			AgentID:       f.Agent.ID,
			ServiceTypeID: f.ServiceType.ID,
		})
		f.PoolValueSvc = mustCreate(t, tx, &domain.ServicePoolValue{
			Name:          "svc-pool-value-1",
			Value:         "v1-svc",
			ServicePoolID: f.ServicePool.ID,
			ServiceID:     &f.Service.ID,
		})
		f.EventSub = mustCreate(t, tx, &domain.EventSubscription{
			SubscriberID: "e2e-subscriber",
			IsActive:     true,
		})
		return nil
	}))

	return f
}

func mustCreate[T any](t *testing.T, tx *gorm.DB, m *T) *T {
	t.Helper()
	require.NoError(t, tx.Create(m).Error, "seed: create %T", m)
	return m
}
