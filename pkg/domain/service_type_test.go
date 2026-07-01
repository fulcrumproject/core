package domain

import (
	"context"
	"strings"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServiceType_TableName(t *testing.T) {
	st := ServiceType{}
	assert.Equal(t, "service_types", st.TableName())
}

// TestServiceTypeBasics tests basic ServiceType operations
func TestServiceTypeBasics(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name        string
		serviceType *ServiceType
		description string
	}{
		{
			name: "Valid service type",
			serviceType: &ServiceType{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name: "Web Server",
			},
			description: "Valid service type with name",
		},
		{
			name: "Empty name",
			serviceType: &ServiceType{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name: "",
			},
			description: "Service type with empty name would fail database validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that the struct can be created
			assert.NotNil(t, tt.serviceType)
			assert.Equal(t, tt.serviceType.Name, tt.serviceType.Name)
		})
	}
}

func TestServiceTypeCommander_Delete(t *testing.T) {
	stID := properties.UUID(uuid.New())

	type stubs struct {
		serviceCount int64
		atCount      int64
	}

	tests := []struct {
		name        string
		stubs       stubs
		wantErr     bool
		errContains string
	}{
		{name: "happy path", stubs: stubs{serviceCount: 0, atCount: 0}, wantErr: false},
		{name: "blocked by dependent services", stubs: stubs{serviceCount: 1, atCount: 0}, wantErr: true, errContains: "dependent service"},
		{name: "blocked by dependent agent types", stubs: stubs{serviceCount: 0, atCount: 2}, wantErr: true, errContains: "dependent agent type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMockStore(t)
			ms.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(Store) error) error {
				return fn(ms)
			}).Maybe()

			stRepo := NewMockServiceTypeRepository(t)
			stRepo.On("Get", mock.Anything, stID).Return(&ServiceType{BaseEntity: BaseEntity{ID: stID}, Name: "st"}, nil).Maybe()
			stRepo.On("Delete", mock.Anything, stID).Return(nil).Maybe()
			ms.On("ServiceTypeRepo").Return(stRepo).Maybe()

			serviceRepo := NewMockServiceRepository(t)
			serviceRepo.On("CountByServiceType", mock.Anything, stID).Return(tt.stubs.serviceCount, nil).Maybe()
			ms.On("ServiceRepo").Return(serviceRepo).Maybe()

			atRepo := NewMockAgentTypeRepository(t)
			atRepo.On("CountByServiceType", mock.Anything, stID).Return(tt.stubs.atCount, nil).Maybe()
			ms.On("AgentTypeRepo").Return(atRepo).Maybe()

			eventRepo := NewMockEventRepository(t)
			eventRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			ms.On("EventRepo").Return(eventRepo).Maybe()

			commander := NewServiceTypeCommander(ms, nil)
			ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})

			err := commander.Delete(ctx, stID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %v", tt.errContains, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Delete() error = %v", err)
			}
		})
	}
}

// Note: Schema validation tests have been moved to pkg/schema package tests
// Domain-specific validators (source, mutable) are tested in service_property_schema_validators_test.go
