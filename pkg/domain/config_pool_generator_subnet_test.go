package domain

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestConfigPoolSubnetGenerator_Allocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	entityID := properties.UUID(uuid.New())
	usedBy := properties.UUID(uuid.New())

	tests := []struct {
		name      string
		config    properties.JSON
		existing  []*ConfigPoolValue
		check     func(t *testing.T, got any)
		wantErr   bool
		errSubstr string
	}{
		{
			name:   "host mode allocates first free, honouring excludeFirst/Last and exclude",
			config: properties.JSON{"cidr": "212.78.11.0/24", "excludeFirst": float64(2), "excludeLast": float64(1), "exclude": []any{"212.78.11.3"}},
			existing: []*ConfigPoolValue{
				{ConfigPoolID: poolID, Value: "212.78.11.2", AgentID: &usedBy},
			},
			check: func(t *testing.T, got any) {
				if got != "212.78.11.4" {
					t.Errorf("expected 212.78.11.4, got %v", got)
				}
			},
		},
		{
			name:   "block mode allocates first free /30 with default host1/host2",
			config: properties.JSON{"cidr": "10.255.1.0/24", "prefix": float64(30)},
			check: func(t *testing.T, got any) {
				m, ok := got.(map[string]any)
				if !ok {
					t.Fatalf("expected map, got %T", got)
				}
				if m["cidr"] != "10.255.1.0/30" || m["host1"] != "10.255.1.1" || m["host2"] != "10.255.1.2" || m["prefix"] != 30 {
					t.Errorf("unexpected block value: %v", m)
				}
			},
		},
		{
			name:   "block mode with empty hosts emits only cidr and prefix",
			config: properties.JSON{"cidr": "10.255.242.0/23", "prefix": float64(24), "hosts": map[string]any{}},
			check: func(t *testing.T, got any) {
				m := got.(map[string]any)
				if len(m) != 2 || m["cidr"] != "10.255.242.0/24" || m["prefix"] != 24 {
					t.Errorf("expected only cidr+prefix, got %v", m)
				}
			},
		},
		{
			name:   "block mode with custom host label",
			config: properties.JSON{"cidr": "10.255.242.0/23", "prefix": float64(24), "hosts": map[string]any{"gateway": float64(1)}},
			check: func(t *testing.T, got any) {
				m := got.(map[string]any)
				if m["cidr"] != "10.255.242.0/24" || m["gateway"] != "10.255.242.1" {
					t.Errorf("unexpected block value: %v", m)
				}
				if _, ok := m["host1"]; ok {
					t.Errorf("did not expect host1 in %v", m)
				}
			},
		},
		{
			name:      "hosts without prefix errors",
			config:    properties.JSON{"cidr": "10.255.1.0/24", "hosts": map[string]any{"gateway": float64(1)}},
			wantErr:   true,
			errSubstr: "requires 'prefix'",
		},
		{
			name:      "hosts offset out of range errors",
			config:    properties.JSON{"cidr": "10.255.1.0/24", "prefix": float64(30), "hosts": map[string]any{"x": float64(4)}},
			wantErr:   true,
			errSubstr: "must be 0..3",
		},
		{
			name:   "block mode skips used /30",
			config: properties.JSON{"cidr": "10.255.1.0/24", "prefix": float64(30)},
			existing: []*ConfigPoolValue{
				{ConfigPoolID: poolID, Value: map[string]any{"cidr": "10.255.1.0/30"}, AgentID: &usedBy},
			},
			check: func(t *testing.T, got any) {
				if got.(map[string]any)["cidr"] != "10.255.1.4/30" {
					t.Errorf("expected 10.255.1.4/30, got %v", got)
				}
			},
		},
		{
			name:      "exhausted subnet errors",
			config:    properties.JSON{"cidr": "10.0.0.0/30", "prefix": float64(30)},
			existing:  []*ConfigPoolValue{{ConfigPoolID: poolID, Value: map[string]any{"cidr": "10.0.0.0/30"}, AgentID: &usedBy}},
			wantErr:   true,
			errSubstr: "subnet exhausted",
		},
		{
			name:      "invalid cidr errors",
			config:    properties.JSON{"cidr": "not-a-cidr"},
			wantErr:   true,
			errSubstr: "is invalid",
		},
		{
			name:      "block mode rejects excludeFirst (host mode only)",
			config:    properties.JSON{"cidr": "10.255.1.0/24", "prefix": float64(30), "excludeFirst": float64(1)},
			wantErr:   true,
			errSubstr: "host mode only",
		},
		{
			name:   "block mode /31 default hosts use RFC 3021 offsets 0 and 1",
			config: properties.JSON{"cidr": "10.0.0.0/24", "prefix": float64(31)},
			check: func(t *testing.T, got any) {
				m := got.(map[string]any)
				if m["cidr"] != "10.0.0.0/31" || m["prefix"] != 31 {
					t.Errorf("unexpected block value: %v", m)
				}
				if m["host1"] != "10.0.0.0" || m["host2"] != "10.0.0.1" {
					t.Errorf("expected host1 10.0.0.0 / host2 10.0.0.1, got %v", m)
				}
			},
		},
		{
			name:   "block mode /32 drops both default hosts",
			config: properties.JSON{"cidr": "10.0.0.0/24", "prefix": float64(32)},
			check: func(t *testing.T, got any) {
				m := got.(map[string]any)
				if len(m) != 2 || m["cidr"] != "10.0.0.0/32" || m["prefix"] != 32 {
					t.Errorf("expected only cidr+prefix, got %v", m)
				}
			},
		},
		{
			name:      "excludeFirst beyond subnet size errors at validation",
			config:    properties.JSON{"cidr": "10.0.0.0/30", "excludeFirst": float64(5)},
			wantErr:   true,
			errSubstr: "excludeFirst",
		},
		{
			name:      "negative excludeFirst errors",
			config:    properties.JSON{"cidr": "10.0.0.0/30", "excludeFirst": float64(-1)},
			wantErr:   true,
			errSubstr: "excludeFirst",
		},
		{
			name:      "excludeFirst plus excludeLast leaving no addresses errors",
			config:    properties.JSON{"cidr": "10.0.0.0/30", "excludeFirst": float64(2), "excludeLast": float64(2)},
			wantErr:   true,
			errSubstr: "excludeLast",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockConfigPoolValueRepository(t)
			repo.On("FindByPool", ctx, poolID).Return(tt.existing, nil).Maybe()
			if !tt.wantErr {
				repo.On("Create", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
					return v.ConfigPoolID == poolID && v.InfrastructureID != nil && *v.InfrastructureID == entityID
				})).Return(nil)
			}

			gen := NewConfigPoolSubnetGenerator(repo, poolID, tt.config)
			got, err := gen.Allocate(ctx, ConfigPoolValueEntityTypeInfrastructure, entityID, "ptpSubnet")

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
			tt.check(t, got)
		})
	}
}

// Release keeps the row: it clears the allocation fields and Updates each value of this
// pool, leaving other pools' values untouched, so a released subnet is never re-issued.
func TestConfigPoolSubnetGenerator_Release(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	otherPool := properties.UUID(uuid.New())
	id1 := properties.UUID(uuid.New())
	id2 := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())
	prop := "subnet"
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

	gen := NewConfigPoolSubnetGenerator(repo, poolID, properties.JSON{})
	if err := gen.Release(ctx, values); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// With no retentionSeconds a freed subnet is reusable immediately: the next allocation
// re-allocates the existing row (Update) instead of minting a new one.
func TestConfigPoolSubnetGenerator_ReusesReleasedValue(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	entityID := properties.UUID(uuid.New())
	releasedAt := time.Now().Add(-time.Hour)

	released := &ConfigPoolValue{
		BaseEntity:   BaseEntity{ID: properties.UUID(uuid.New())},
		ConfigPoolID: poolID,
		Value:        "212.78.11.2",
		ReleasedAt:   &releasedAt,
	}
	repo := NewMockConfigPoolValueRepository(t)
	repo.On("FindByPool", ctx, poolID).Return([]*ConfigPoolValue{released}, nil)
	repo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
		return v.ID == released.ID && v.IsAllocated() && v.ReleasedAt == nil
	})).Return(nil)

	gen := NewConfigPoolSubnetGenerator(repo, poolID, properties.JSON{"cidr": "212.78.11.0/24", "excludeFirst": float64(2)})
	got, err := gen.Allocate(ctx, ConfigPoolValueEntityTypeInfrastructure, entityID, "ip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "212.78.11.2" {
		t.Errorf("expected freed 212.78.11.2 to be reused, got %v", got)
	}
}

// Within the retentionSeconds cooldown a freed subnet stays reserved and the next free
// value is minted instead.
func TestConfigPoolSubnetGenerator_RetentionHoldsValue(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	entityID := properties.UUID(uuid.New())
	releasedAt := time.Now()

	released := &ConfigPoolValue{
		BaseEntity:   BaseEntity{ID: properties.UUID(uuid.New())},
		ConfigPoolID: poolID,
		Value:        "212.78.11.2",
		ReleasedAt:   &releasedAt,
	}
	repo := NewMockConfigPoolValueRepository(t)
	repo.On("FindByPool", ctx, poolID).Return([]*ConfigPoolValue{released}, nil)
	repo.On("Create", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
		return v.Value == "212.78.11.3"
	})).Return(nil)

	gen := NewConfigPoolSubnetGenerator(repo, poolID, properties.JSON{"cidr": "212.78.11.0/24", "excludeFirst": float64(2), "retentionSeconds": float64(3600)})
	got, err := gen.Allocate(ctx, ConfigPoolValueEntityTypeInfrastructure, entityID, "ip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "212.78.11.3" {
		t.Errorf("expected 212.78.11.3 (212.78.11.2 still cooling), got %v", got)
	}
}

// neverReallocate keeps a freed subnet out of circulation forever: even past any
// cooldown the freed value is skipped and the next free one is minted.
func TestConfigPoolSubnetGenerator_NeverReallocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	entityID := properties.UUID(uuid.New())
	releasedAt := time.Now().Add(-100 * time.Hour)

	released := &ConfigPoolValue{
		BaseEntity:   BaseEntity{ID: properties.UUID(uuid.New())},
		ConfigPoolID: poolID,
		Value:        "212.78.11.2",
		ReleasedAt:   &releasedAt,
	}
	repo := NewMockConfigPoolValueRepository(t)
	repo.On("FindByPool", ctx, poolID).Return([]*ConfigPoolValue{released}, nil)
	repo.On("Create", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
		return v.Value == "212.78.11.3"
	})).Return(nil)

	gen := NewConfigPoolSubnetGenerator(repo, poolID, properties.JSON{"cidr": "212.78.11.0/24", "excludeFirst": float64(2), "neverReallocate": true})
	got, err := gen.Allocate(ctx, ConfigPoolValueEntityTypeInfrastructure, entityID, "ip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "212.78.11.3" {
		t.Errorf("expected 212.78.11.3 (212.78.11.2 never reallocated), got %v", got)
	}
}
