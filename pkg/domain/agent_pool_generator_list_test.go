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

func TestAgentPoolListGenerator_Allocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())

	tests := []struct {
		name      string
		setupRepo func(*MockAgentPoolValueRepository)
		wantValue any
		wantErr   bool
		errSubstr string
	}{
		{
			name: "happy path",
			setupRepo: func(repo *MockAgentPoolValueRepository) {
				values := []*AgentPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolID, Value: "10.0.0.1"},
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolID, Value: "10.0.0.2"},
				}
				repo.On("FindAvailable", ctx, poolID).Return(values, nil)
				repo.On("Update", ctx, mock.MatchedBy(func(v *AgentPoolValue) bool {
					return v.AgentID != nil && *v.AgentID == agentID &&
						v.PropertyName != nil && *v.PropertyName == "propA" &&
						v.AllocatedAt != nil
				})).Return(nil)
			},
			wantValue: "10.0.0.1",
		},
		{
			name: "no available values",
			setupRepo: func(repo *MockAgentPoolValueRepository) {
				repo.On("FindAvailable", ctx, poolID).Return([]*AgentPoolValue{}, nil)
			},
			wantErr:   true,
			errSubstr: "no available values",
		},
		{
			name: "find available errors",
			setupRepo: func(repo *MockAgentPoolValueRepository) {
				repo.On("FindAvailable", ctx, poolID).Return(nil, errors.New("db boom"))
			},
			wantErr:   true,
			errSubstr: "db boom",
		},
		{
			name: "update errors",
			setupRepo: func(repo *MockAgentPoolValueRepository) {
				values := []*AgentPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolID, Value: "x"},
				}
				repo.On("FindAvailable", ctx, poolID).Return(values, nil)
				repo.On("Update", ctx, mock.AnythingOfType("*domain.AgentPoolValue")).Return(errors.New("update boom"))
			},
			wantErr:   true,
			errSubstr: "update boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockAgentPoolValueRepository(t)
			tt.setupRepo(repo)

			gen := NewAgentPoolListGenerator(repo, poolID)
			got, err := gen.Allocate(ctx, agentID, "propA")

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

func TestAgentPoolListGenerator_Release(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	otherPoolID := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())
	now := time.Now()

	tests := []struct {
		name      string
		values    []*AgentPoolValue
		setupRepo func(*MockAgentPoolValueRepository)
		wantErr   bool
		errSubstr string
	}{
		{
			name: "releases only values from this pool",
			values: []*AgentPoolValue{
				{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolID, AgentID: &agentID, AllocatedAt: &now, Value: "a"},
				{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: otherPoolID, AgentID: &agentID, AllocatedAt: &now, Value: "b"},
				{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolID, AgentID: &agentID, AllocatedAt: &now, Value: "c"},
			},
			setupRepo: func(repo *MockAgentPoolValueRepository) {
				repo.On("Update", ctx, mock.MatchedBy(func(v *AgentPoolValue) bool {
					return v.AgentPoolID == poolID && v.AgentID == nil && v.PropertyName == nil && v.AllocatedAt == nil
				})).Return(nil).Twice()
			},
		},
		{
			name:      "no allocations — no updates",
			values:    []*AgentPoolValue{},
			setupRepo: func(repo *MockAgentPoolValueRepository) {},
		},
		{
			name: "update errors",
			values: []*AgentPoolValue{
				{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, AgentPoolID: poolID, AgentID: &agentID, AllocatedAt: &now, Value: "x"},
			},
			setupRepo: func(repo *MockAgentPoolValueRepository) {
				repo.On("Update", ctx, mock.AnythingOfType("*domain.AgentPoolValue")).Return(errors.New("update boom"))
			},
			wantErr:   true,
			errSubstr: "update boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockAgentPoolValueRepository(t)
			tt.setupRepo(repo)

			gen := NewAgentPoolListGenerator(repo, poolID)
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
