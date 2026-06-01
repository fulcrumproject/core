package domain

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestConfigPoolSubnetGenerator_Allocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	entityID := properties.UUID(uuid.New())

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
				{ConfigPoolID: poolID, Value: "212.78.11.2"},
			},
			check: func(t *testing.T, got any) {
				if got != "212.78.11.4" {
					t.Errorf("expected 212.78.11.4, got %v", got)
				}
			},
		},
		{
			name:   "block mode allocates first free /30 as JSON with derived hosts",
			config: properties.JSON{"cidr": "10.255.1.0/24", "prefix": float64(30)},
			check: func(t *testing.T, got any) {
				m, ok := got.(map[string]any)
				if !ok {
					t.Fatalf("expected map, got %T", got)
				}
				if m["cidr"] != "10.255.1.0/30" || m["fulcrumIp"] != "10.255.1.1" || m["cspIp"] != "10.255.1.2" || m["prefix"] != 30 {
					t.Errorf("unexpected block value: %v", m)
				}
			},
		},
		{
			name:   "block mode skips used /30",
			config: properties.JSON{"cidr": "10.255.1.0/24", "prefix": float64(30)},
			existing: []*ConfigPoolValue{
				{ConfigPoolID: poolID, Value: map[string]any{"cidr": "10.255.1.0/30"}},
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
			existing:  []*ConfigPoolValue{{ConfigPoolID: poolID, Value: map[string]any{"cidr": "10.0.0.0/30"}}},
			wantErr:   true,
			errSubstr: "subnet exhausted",
		},
		{
			name:      "invalid cidr errors",
			config:    properties.JSON{"cidr": "not-a-cidr"},
			wantErr:   true,
			errSubstr: "is invalid",
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
