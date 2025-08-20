package database

import (
	"context"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
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
		// Create VM service type with property schema
		vmSchema := &schema.CustomSchema{
			"cpu": schema.PropertyDefinition{
				Type:     schema.TypeInteger,
				Label:    "CPU Cores",
				Required: true,
				Validators: []schema.ValidatorDefinition{
					{Type: schema.ValidatorEnum, Value: []any{1, 2, 4, 8, 16, 32}},
				},
			},
			"memory": schema.PropertyDefinition{
				Type:     schema.TypeInteger,
				Label:    "Memory (MB)",
				Required: true,
				Validators: []schema.ValidatorDefinition{
					{Type: schema.ValidatorEnum, Value: []any{512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}},
				},
			},
			"disk": schema.PropertyDefinition{
				Type:     schema.TypeInteger,
				Label:    "Disk (GB)",
				Required: true,
				Validators: []schema.ValidatorDefinition{
					{Type: schema.ValidatorEnum, Value: []any{8, 16, 32, 64, 128, 256, 512, 1024}},
				},
			},
			"image": schema.PropertyDefinition{
				Type:     schema.TypeString,
				Label:    "Image",
				Required: true,
			},
		}

		vmServiceType = domain.ServiceType{
			BaseEntity: domain.BaseEntity{
				ID: vmServiceTypeID,
			},
			Name:           "vm",
			PropertySchema: vmSchema,
		}
		if err := serviceTypeRepo.Create(ctx, &vmServiceType); err != nil {
			return err
		}
	}

	// Create kubernetes service type if needed
	var kubernetesServiceType domain.ServiceType
	kubernetesServiceTypeID := uuid.MustParse("019760cf-94bd-7859-bea9-62d945ec5ce3")
	exists, err = serviceTypeRepo.Exists(ctx, kubernetesServiceTypeID)
	if err != nil {
		return err
	}
	if !exists {
		// Create Kubernetes cluster service type with property schema
		kubernetesSchema := &schema.CustomSchema{
			"nodes": schema.PropertyDefinition{
				Type:     schema.TypeArray,
				Label:    "Cluster Nodes",
				Required: true,
				Items: &schema.PropertyDefinition{
					Type: schema.TypeObject,
					Properties: map[string]schema.PropertyDefinition{
						"id": {
							Type:     schema.TypeString,
							Label:    "Node ID",
							Required: true,
							Validators: []schema.ValidatorDefinition{
								{Type: schema.ValidatorMinLength, Value: 1},
								{Type: schema.ValidatorMaxLength, Value: 50},
							},
						},
						"size": {
							Type:     schema.TypeString,
							Label:    "Node Size",
							Required: true,
							Validators: []schema.ValidatorDefinition{
								{Type: schema.ValidatorEnum, Value: []any{"s1", "s2", "s4"}},
							},
						},
						"status": {
							Type:     schema.TypeString,
							Label:    "Node Status",
							Required: true,
							Validators: []schema.ValidatorDefinition{
								{Type: schema.ValidatorEnum, Value: []any{"On", "Off"}},
							},
						},
					},
				},
				Validators: []schema.ValidatorDefinition{
					{Type: schema.ValidatorMinItems, Value: 1},
					{Type: schema.ValidatorMaxItems, Value: 100},
				},
			},
		}

		kubernetesServiceType = domain.ServiceType{
			BaseEntity: domain.BaseEntity{
				ID: kubernetesServiceTypeID,
			},
			Name:           "kubernetes-cluster",
			PropertySchema: kubernetesSchema,
		}
		if err := serviceTypeRepo.Create(ctx, &kubernetesServiceType); err != nil {
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

	// Create kubernetes agent type if needed
	kubernetesAgentTypeID := uuid.MustParse("019760d0-0f14-7853-9cad-b9ab8ce950f4")
	exists, err = agentTypeRepo.Exists(ctx, kubernetesAgentTypeID)
	if err != nil {
		return err
	}
	if !exists {
		kubernetesAgentType := &domain.AgentType{
			BaseEntity: domain.BaseEntity{
				ID: kubernetesAgentTypeID,
			},
			Name: "kubernetes",
			ServiceTypes: []domain.ServiceType{
				kubernetesServiceType,
			},
		}
		if err := agentTypeRepo.Create(ctx, kubernetesAgentType); err != nil {
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
			Role:        auth.RoleAdmin,
			ExpireAt:    time.Now().AddDate(10, 0, 0), // 10 years in the future
		}
		if err := tokenRepo.Create(ctx, adminToken); err != nil {
			return err
		}
	}

	return nil
}
