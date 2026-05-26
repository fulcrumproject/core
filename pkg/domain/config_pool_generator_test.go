package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

func TestDefaultConfigPoolGeneratorFactory_CreateGenerator(t *testing.T) {
	tests := []struct {
		name          string
		generatorType PoolGeneratorType
		wantType      any
		wantErr       bool
		errSubstr     string
	}{
		{
			name:          "list → ConfigPoolListGenerator",
			generatorType: PoolGeneratorList,
			wantType:      (*ConfigPoolListGenerator)(nil),
		},
		{
			name:          "subnet is not yet supported → error",
			generatorType: PoolGeneratorSubnet,
			wantErr:       true,
			errSubstr:     "unsupported config pool generator type",
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
				BaseEntity:    BaseEntity{ID: properties.UUID(uuid.New())},
				GeneratorType: tt.generatorType,
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
			if _, ok := gen.(*ConfigPoolListGenerator); !ok && tt.generatorType == PoolGeneratorList {
				t.Errorf("expected *ConfigPoolListGenerator, got %T", gen)
			}
		})
	}
}
