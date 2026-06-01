package domain

import (
	"reflect"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

func TestDefaultConfigPoolGeneratorFactory_CreateGenerator(t *testing.T) {
	tests := []struct {
		name            string
		generatorType   PoolGeneratorType
		generatorConfig *properties.JSON
		wantType        any
		wantErr         bool
		errSubstr       string
	}{
		{
			name:          "list → ConfigPoolListGenerator",
			generatorType: PoolGeneratorList,
			wantType:      (*ConfigPoolListGenerator)(nil),
		},
		{
			name:            "range → ConfigPoolRangeGenerator",
			generatorType:   PoolGeneratorRange,
			generatorConfig: &properties.JSON{"min": float64(1), "max": float64(10)},
			wantType:        (*ConfigPoolRangeGenerator)(nil),
		},
		{
			name:          "range without config → error",
			generatorType: PoolGeneratorRange,
			wantErr:       true,
			errSubstr:     "missing generatorConfig",
		},
		{
			name:            "subnet → ConfigPoolSubnetGenerator",
			generatorType:   PoolGeneratorSubnet,
			generatorConfig: &properties.JSON{"cidr": "10.0.0.0/24"},
			wantType:        (*ConfigPoolSubnetGenerator)(nil),
		},
		{
			name:          "subnet without config → error",
			generatorType: PoolGeneratorSubnet,
			wantErr:       true,
			errSubstr:     "missing generatorConfig",
		},
		{
			name:          "unknown type → error",
			generatorType: PoolGeneratorType("bogus"),
			wantErr:       true,
			errSubstr:     "unsupported config pool generator type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockConfigPoolValueRepository(t)
			factory := NewDefaultConfigPoolGeneratorFactory(repo)

			pool := &ConfigPool{
				BaseEntity:      BaseEntity{ID: properties.UUID(uuid.New())},
				GeneratorType:   tt.generatorType,
				GeneratorConfig: tt.generatorConfig,
			}

			gen, err := factory.CreateGenerator(pool)
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
			if reflect.TypeOf(gen) != reflect.TypeOf(tt.wantType) {
				t.Errorf("expected %T, got %T", tt.wantType, gen)
			}
		})
	}
}
