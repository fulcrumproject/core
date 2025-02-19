package database

import (
	"context"

	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {

	agentTypeRepo := NewAgentTypeRepository(db)
	serviceTypeRepo := NewServiceTypeRepository(db)

	ctx := context.Background()

	na, err := agentTypeRepo.Count(ctx)
	if err != nil {
		return err
	}

	ns, err := serviceTypeRepo.Count(ctx)
	if err != nil {
		return err
	}

	if na > 0 || ns > 0 {
		return nil
	}

	// Fixed UUIDs for default types
	dummyAgentTypeID := domain.UUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	vmServiceTypeID := domain.UUID(uuid.MustParse("22222222-2222-2222-2222-222222222222"))

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

	return nil
}
