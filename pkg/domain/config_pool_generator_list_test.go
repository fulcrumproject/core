package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestConfigPoolListGenerator_Allocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())
	infraID := properties.UUID(uuid.New())

	tests := []struct {
		name       string
		entityType ConfigPoolValueEntityType
		entityID   properties.UUID
		setupRepo  func(*MockConfigPoolValueRepository)
		wantValue  any
		wantErr    bool
		errSubstr  string
	}{
		{
			name:       "happy path agent",
			entityType: ConfigPoolValueEntityTypeAgent,
			entityID:   agentID,
			setupRepo: func(repo *MockConfigPoolValueRepository) {
				values := []*ConfigPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, Value: "10.0.0.1"},
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, Value: "10.0.0.2"},
				}
				repo.On("FindAvailable", ctx, poolID).Return(values, nil)
				repo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
					return v.AgentID != nil && *v.AgentID == agentID && v.InfrastructureID == nil &&
						v.PropertyName != nil && *v.PropertyName == "propA" &&
						v.AllocatedAt != nil
				})).Return(nil)
			},
			wantValue: "10.0.0.1",
		},
		{
			name:       "happy path infrastructure",
			entityType: ConfigPoolValueEntityTypeInfrastructure,
			entityID:   infraID,
			setupRepo: func(repo *MockConfigPoolValueRepository) {
				values := []*ConfigPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, Value: "10.0.0.1"},
				}
				repo.On("FindAvailable", ctx, poolID).Return(values, nil)
				repo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
					return v.InfrastructureID != nil && *v.InfrastructureID == infraID && v.AgentID == nil &&
						v.PropertyName != nil && *v.PropertyName == "propA" &&
						v.AllocatedAt != nil
				})).Return(nil)
			},
			wantValue: "10.0.0.1",
		},
		{
			name:       "no available values",
			entityType: ConfigPoolValueEntityTypeAgent,
			entityID:   agentID,
			setupRepo: func(repo *MockConfigPoolValueRepository) {
				repo.On("FindAvailable", ctx, poolID).Return([]*ConfigPoolValue{}, nil)
			},
			wantErr:   true,
			errSubstr: "no available values",
		},
		{
			name:       "find available errors",
			entityType: ConfigPoolValueEntityTypeAgent,
			entityID:   agentID,
			setupRepo: func(repo *MockConfigPoolValueRepository) {
				repo.On("FindAvailable", ctx, poolID).Return(nil, errors.New("db boom"))
			},
			wantErr:   true,
			errSubstr: "db boom",
		},
		{
			name:       "update errors",
			entityType: ConfigPoolValueEntityTypeAgent,
			entityID:   agentID,
			setupRepo: func(repo *MockConfigPoolValueRepository) {
				values := []*ConfigPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, Value: "x"},
				}
				repo.On("FindAvailable", ctx, poolID).Return(values, nil)
				repo.On("Update", ctx, mock.AnythingOfType("*domain.ConfigPoolValue")).Return(errors.New("update boom"))
			},
			wantErr:   true,
			errSubstr: "update boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockConfigPoolValueRepository(t)
			tt.setupRepo(repo)

			gen := NewConfigPoolListGenerator(repo, poolID)
			got, err := gen.Allocate(ctx, tt.entityType, tt.entityID, "propA")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !stringContains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantValue {
				t.Errorf("expected value=%v, got %v", tt.wantValue, got)
			}
		})
	}
}

func TestConfigPoolListGenerator_Release(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	otherPoolID := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())
	now := time.Now()

	tests := []struct {
		name      string
		values    []*ConfigPoolValue
		setupRepo func(*MockConfigPoolValueRepository)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "releases only values from this pool",
			values: []*ConfigPoolValue{
				{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, AgentID: &agentID, AllocatedAt: &now, Value: "a"},
				{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: otherPoolID, AgentID: &agentID, AllocatedAt: &now, Value: "b"},
				{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, AgentID: &agentID, AllocatedAt: &now, Value: "c"},
			},
			setupRepo: func(repo *MockConfigPoolValueRepository) {
				repo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
					return v.ConfigPoolID == poolID && v.AgentID == nil && v.PropertyName == nil && v.AllocatedAt == nil
				})).Return(nil).Twice()
			},
		},
		{
			name:      "no allocations — no updates",
			values:    []*ConfigPoolValue{},
			setupRepo: func(repo *MockConfigPoolValueRepository) {},
		},
		{
			name: "update errors",
			values: []*ConfigPoolValue{
				{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, AgentID: &agentID, AllocatedAt: &now, Value: "x"},
			},
			setupRepo: func(repo *MockConfigPoolValueRepository) {
				repo.On("Update", ctx, mock.AnythingOfType("*domain.ConfigPoolValue")).Return(errors.New("update boom"))
			},
			wantErr:   true,
			errSubstr: "update boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockConfigPoolValueRepository(t)
			tt.setupRepo(repo)

			gen := NewConfigPoolListGenerator(repo, poolID)
			err := gen.Release(ctx, tt.values)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !stringContains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
