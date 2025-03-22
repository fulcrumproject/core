package database

import (
	"context"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {

	agentTypeRepo := NewAgentTypeRepository(db)
	serviceTypeRepo := NewServiceTypeRepository(db)
	metricTypeRepo := NewMetricTypeRepository(db)
	tokenRepo := NewTokenRepository(db)

	ctx := context.Background()

	na, err := agentTypeRepo.Count(ctx)
	if err != nil {
		return err
	}

	ns, err := serviceTypeRepo.Count(ctx)
	if err != nil {
		return err
	}

	nm, err := metricTypeRepo.Count(ctx)
	if err != nil {
		return err
	}

	nt, err := tokenRepo.Count(ctx)
	if err != nil {
		return err
	}

	if na > 0 || ns > 0 || nm > 0 || nt > 0 {
		return nil
	}

	// Create default entity types
	// Fixed UUIDs for default types
	dummyAgentTypeID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	vmServiceTypeID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	// Create vm service type
	vmServiceType := domain.ServiceType{
		BaseEntity: domain.BaseEntity{
			ID: vmServiceTypeID,
		},
		Name: "vm",
	}

	// Create dummy agent type
	dummyAgentType := &domain.AgentType{
		BaseEntity: domain.BaseEntity{
			ID: dummyAgentTypeID,
		},
		Name: "dummy",
		ServiceTypes: []domain.ServiceType{
			vmServiceType,
		},
	}
	if err := agentTypeRepo.Create(ctx, dummyAgentType); err != nil {
		return err
	}

	// Create metric types
	metricTypes := []domain.MetricType{
		{Name: "vm.cpu.usage", EntityType: domain.MetricEntityTypeResource},
		{Name: "vm.memory.usage", EntityType: domain.MetricEntityTypeResource},
		{Name: "vm.disk.usage", EntityType: domain.MetricEntityTypeResource},
		{Name: "vm.network.throughput", EntityType: domain.MetricEntityTypeResource},
	}

	// Create the metric types
	for i := range metricTypes {
		if err := metricTypeRepo.Create(ctx, &metricTypes[i]); err != nil {
			return err
		}
	}

	// Create a default admin token for tests
	adminTokenID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	// Use a fixed token value for tests
	const adminTokenValue = "admin-test-token"

	adminToken := &domain.Token{
		BaseEntity: domain.BaseEntity{
			ID: adminTokenID,
		},
		Name:        "Admin Test Token",
		PlainValue:  adminTokenValue,
		HashedValue: domain.HashTokenValue(adminTokenValue),
		Role:        domain.RoleFulcrumAdmin,
		ExpireAt:    time.Now().AddDate(10, 0, 0), // 10 years in the future
	}
	if err := tokenRepo.Create(ctx, adminToken); err != nil {
		return err
	}

	return nil
}
