package domain

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestConfigPoolRangeGenerator_Allocate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	entityID := properties.UUID(uuid.New())

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
			name:   "skips already-used values",
			config: properties.JSON{"min": float64(65000), "max": float64(65535)},
			existing: []*ConfigPoolValue{
				{ConfigPoolID: poolID, Value: float64(65000)},
				{ConfigPoolID: poolID, Value: float64(65001)},
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
				{ConfigPoolID: poolID, Value: float64(65000)},
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

func TestConfigPoolRangeGenerator_Release(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	otherPool := properties.UUID(uuid.New())
	id1 := properties.UUID(uuid.New())
	id2 := properties.UUID(uuid.New())

	values := []*ConfigPoolValue{
		{BaseEntity: BaseEntity{ID: id1}, ConfigPoolID: poolID},
		{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: otherPool},
		{BaseEntity: BaseEntity{ID: id2}, ConfigPoolID: poolID},
	}

	repo := NewMockConfigPoolValueRepository(t)
	repo.On("DeleteByIDs", ctx, mock.MatchedBy(func(ids []properties.UUID) bool {
		return len(ids) == 2 && ids[0] == id1 && ids[1] == id2
	})).Return(nil)

	gen := NewConfigPoolRangeGenerator(repo, poolID, properties.JSON{})
	if err := gen.Release(ctx, values); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
