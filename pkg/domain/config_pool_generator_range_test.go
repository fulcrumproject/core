package domain

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestConfigPoolRangeGenerator_Allocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	entityID := properties.UUID(uuid.New())
	usedBy := properties.UUID(uuid.New())

	tests := []struct {
		name      string
		config    properties.JSON
		existing  []*ConfigPoolValue
		wantValue any
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "empty pool allocates min",
			config:    properties.JSON{"min": float64(65000), "max": float64(65535)},
			wantValue: 65000,
		},
		{
			name:   "skips allocated values",
			config: properties.JSON{"min": float64(65000), "max": float64(65535)},
			existing: []*ConfigPoolValue{
				{ConfigPoolID: poolID, Value: float64(65000), AgentID: &usedBy},
				{ConfigPoolID: poolID, Value: float64(65001), AgentID: &usedBy},
			},
			wantValue: 65002,
		},
		{
			name:      "skips excluded values",
			config:    properties.JSON{"min": float64(65500), "max": float64(65535), "exclude": []any{float64(65500)}},
			wantValue: 65501,
		},
		{
			name:   "exhausted range errors",
			config: properties.JSON{"min": float64(65000), "max": float64(65000)},
			existing: []*ConfigPoolValue{
				{ConfigPoolID: poolID, Value: float64(65000), AgentID: &usedBy},
			},
			wantErr:   true,
			errSubstr: "range exhausted",
		},
		{
			name:      "invalid config errors",
			config:    properties.JSON{"min": float64(10)},
			wantErr:   true,
			errSubstr: "requires integer 'max'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockConfigPoolValueRepository(t)
			repo.On("FindByPool", ctx, poolID).Return(tt.existing, nil).Maybe()
			if !tt.wantErr {
				repo.On("Create", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
					n, ok := toInt(v.Value)
					return ok && n == tt.wantValue.(int) && v.ConfigPoolID == poolID &&
						v.InfrastructureID != nil && *v.InfrastructureID == entityID &&
						v.PropertyName != nil && *v.PropertyName == "asn"
				})).Return(nil)
			}

			gen := NewConfigPoolRangeGenerator(repo, poolID, tt.config)
			got, err := gen.Allocate(ctx, ConfigPoolValueEntityTypeInfrastructure, entityID, "asn")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if !stringContains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantValue {
				t.Errorf("expected value %v, got %v", tt.wantValue, got)
			}
		})
	}
}

// Release keeps the row: it clears the allocation fields and Updates each value of
// this pool, so the value stays "used" and is never re-allocated. Values of other
// pools are left untouched.
func TestConfigPoolRangeGenerator_Release(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	otherPool := properties.UUID(uuid.New())
	id1 := properties.UUID(uuid.New())
	id2 := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())
	prop := "asn"
	now := time.Now()

	values := []*ConfigPoolValue{
		{BaseEntity: BaseEntity{ID: id1}, ConfigPoolID: poolID, AgentID: &agentID, PropertyName: &prop, AllocatedAt: &now},
		{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: otherPool, AgentID: &agentID, PropertyName: &prop, AllocatedAt: &now},
		{BaseEntity: BaseEntity{ID: id2}, ConfigPoolID: poolID, AgentID: &agentID, PropertyName: &prop, AllocatedAt: &now},
	}

	repo := NewMockConfigPoolValueRepository(t)
	repo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
		return (v.ID == id1 || v.ID == id2) && !v.IsAllocated() &&
			v.AgentID == nil && v.InfrastructureID == nil && v.AllocatedAt == nil && v.PropertyName == nil
	})).Return(nil).Twice()

	gen := NewConfigPoolRangeGenerator(repo, poolID, properties.JSON{})
	if err := gen.Release(ctx, values); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// With no retentionSeconds, a freed value is reusable immediately: the next allocation
// re-allocates the existing row in place (Update) rather than minting a new one.
func TestConfigPoolRangeGenerator_ReusesReleasedValue(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())
	releasedAt := time.Now().Add(-time.Hour)

	released := &ConfigPoolValue{
		BaseEntity:   BaseEntity{ID: properties.UUID(uuid.New())},
		ConfigPoolID: poolID,
		Value:        float64(1),
		ReleasedAt:   &releasedAt,
	}

	repo := NewMockConfigPoolValueRepository(t)
	repo.On("FindByPool", ctx, poolID).Return([]*ConfigPoolValue{released}, nil)
	repo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
		return v.ID == released.ID && v.IsAllocated() &&
			v.AgentID != nil && *v.AgentID == agentID && v.ReleasedAt == nil
	})).Return(nil)

	gen := NewConfigPoolRangeGenerator(repo, poolID, properties.JSON{"min": float64(1), "max": float64(3)})
	got, err := gen.Allocate(ctx, ConfigPoolValueEntityTypeAgent, agentID, "asn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n, ok := toInt(got); !ok || n != 1 {
		t.Errorf("expected freed value 1 to be reused, got %v", got)
	}
}

// retentionSeconds blocks reuse until the cooldown elapses: within the window the freed
// value is skipped and a new one is minted; past it the freed value is re-allocated.
func TestConfigPoolRangeGenerator_RetentionCooldown(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())
	config := properties.JSON{"min": float64(1), "max": float64(3), "retentionSeconds": float64(3600)}

	t.Run("within cooldown mints next value", func(t *testing.T) {
		releasedAt := time.Now()
		released := &ConfigPoolValue{
			BaseEntity:   BaseEntity{ID: properties.UUID(uuid.New())},
			ConfigPoolID: poolID,
			Value:        float64(1),
			ReleasedAt:   &releasedAt,
		}
		repo := NewMockConfigPoolValueRepository(t)
		repo.On("FindByPool", ctx, poolID).Return([]*ConfigPoolValue{released}, nil)
		repo.On("Create", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
			n, ok := toInt(v.Value)
			return ok && n == 2
		})).Return(nil)

		gen := NewConfigPoolRangeGenerator(repo, poolID, config)
		got, err := gen.Allocate(ctx, ConfigPoolValueEntityTypeAgent, agentID, "asn")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n, _ := toInt(got); n != 2 {
			t.Errorf("expected 2 (value 1 still cooling), got %v", got)
		}
	})

	t.Run("after cooldown reuses freed value", func(t *testing.T) {
		releasedAt := time.Now().Add(-2 * time.Hour)
		released := &ConfigPoolValue{
			BaseEntity:   BaseEntity{ID: properties.UUID(uuid.New())},
			ConfigPoolID: poolID,
			Value:        float64(1),
			ReleasedAt:   &releasedAt,
		}
		repo := NewMockConfigPoolValueRepository(t)
		repo.On("FindByPool", ctx, poolID).Return([]*ConfigPoolValue{released}, nil)
		repo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
			return v.ID == released.ID && v.IsAllocated() && v.ReleasedAt == nil
		})).Return(nil)

		gen := NewConfigPoolRangeGenerator(repo, poolID, config)
		got, err := gen.Allocate(ctx, ConfigPoolValueEntityTypeAgent, agentID, "asn")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n, _ := toInt(got); n != 1 {
			t.Errorf("expected reused 1 after cooldown, got %v", got)
		}
	})
}
