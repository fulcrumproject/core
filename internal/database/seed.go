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

	// Create vm service type if needed
	var vmServiceType domain.ServiceType
	vmServiceTypeID := uuid.MustParse("0195c3c8-69e5-7806-9598-8523c01ea54f")
	exists, err := serviceTypeRepo.Exists(ctx, vmServiceTypeID)
	if err != nil {
		return err
	}
	if !exists {
		vmServiceType = domain.ServiceType{
			BaseEntity: domain.BaseEntity{
				ID: vmServiceTypeID,
			},
			Name: "vm",
		}
		if err := serviceTypeRepo.Create(ctx, &vmServiceType); err != nil {
			return err
		}
	}

	// Create dummy agent type if needed
	dummyAgentTypeID := uuid.MustParse("0195c3c6-4c7d-7e3c-b481-f276e17a7bec")
	exists, err = agentTypeRepo.Exists(ctx, dummyAgentTypeID)
	if err != nil {
		return err
	}
	if !exists {
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
	}

	// Create test metric types if needed
	metricTypes := []domain.MetricType{
		{BaseEntity: domain.BaseEntity{ID: uuid.MustParse("0195c3c9-a211-753b-86d0-be343ec40df4")}, Name: "vm.cpu.usage", EntityType: domain.MetricEntityTypeResource},
		{BaseEntity: domain.BaseEntity{ID: uuid.MustParse("0195c3c9-fb93-717e-8e4d-94247359e35c")}, Name: "vm.memory.usage", EntityType: domain.MetricEntityTypeResource},
		{BaseEntity: domain.BaseEntity{ID: uuid.MustParse("0195c3ca-2c6e-771c-8250-3a5dabaaceee")}, Name: "vm.disk.usage", EntityType: domain.MetricEntityTypeResource},
		{BaseEntity: domain.BaseEntity{ID: uuid.MustParse("0195c3ca-6334-74fd-a230-a64bf1d4f376")}, Name: "vm.network.throughput", EntityType: domain.MetricEntityTypeResource},
	}
	for i, mt := range metricTypes {
		exists, err := metricTypeRepo.Exists(ctx, mt.ID)
		if err != nil {
			return err
		}
		if !exists {
			if err := metricTypeRepo.Create(ctx, &metricTypes[i]); err != nil {
				return err
			}
		}
	}

	// Create a default admin token for tests
	adminTokenID := uuid.MustParse("0195c3cc-a21a-7474-a214-f6fd48d4609b")
	exists, err = tokenRepo.Exists(ctx, adminTokenID)
	if err != nil {
		return err
	}
	if !exists {
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
	}

	return nil
}
