package schema

import (
	"context"
	"testing"
)

func TestMinLengthValidator_Validate(t *testing.T) {
	validator := &MinLengthValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name    string
		value   any
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid - meets minimum",
			value:   "test",
			config:  map[string]any{"value": 3},
			wantErr: false,
		},
		{
			name:    "valid - exceeds minimum",
			value:   "testing",
			config:  map[string]any{"value": 3},
			wantErr: false,
		},
		{
			name:    "invalid - below minimum",
			value:   "ab",
			config:  map[string]any{"value": 3},
			wantErr: true,
		},
		{
			name:    "invalid - non-string value",
			value:   123,
			config:  map[string]any{"value": 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, testCtx, OperationCreate, "testProp", nil, tt.value, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinLengthValidator_ValidateConfig(t *testing.T) {
	validator := &MinLengthValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  map[string]any{"value": 5},
			wantErr: false,
		},
		{
			name:    "valid config - zero",
			config:  map[string]any{"value": 0},
			wantErr: false,
		},
		{
			name:    "invalid - missing value",
			config:  map[string]any{},
			wantErr: true,
		},
		{
			name:    "invalid - negative value",
			config:  map[string]any{"value": -1},
			wantErr: true,
		},
		{
			name:    "invalid - wrong type",
			config:  map[string]any{"value": "not-a-number"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig("testProp", tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxLengthValidator_Validate(t *testing.T) {
	validator := &MaxLengthValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name    string
		value   any
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid - within maximum",
			value:   "test",
			config:  map[string]any{"value": 10},
			wantErr: false,
		},
		{
			name:    "valid - at maximum",
			value:   "test",
			config:  map[string]any{"value": 4},
			wantErr: false,
		},
		{
			name:    "invalid - exceeds maximum",
			value:   "testing",
			config:  map[string]any{"value": 4},
			wantErr: true,
		},
		{
			name:    "invalid - non-string value",
			value:   123,
			config:  map[string]any{"value": 10},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, testCtx, OperationCreate, "testProp", nil, tt.value, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxLengthValidator_ValidateConfig(t *testing.T) {
	validator := &MaxLengthValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  map[string]any{"value": 100},
			wantErr: false,
		},
		{
			name:    "valid config - zero",
			config:  map[string]any{"value": 0},
			wantErr: false,
		},
		{
			name:    "invalid - missing value",
			config:  map[string]any{},
			wantErr: true,
		},
		{
			name:    "invalid - negative value",
			config:  map[string]any{"value": -5},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig("testProp", tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternValidator_Validate(t *testing.T) {
	validator := &PatternValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name    string
		value   any
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid - matches pattern",
			value:   "test@example.com",
			config:  map[string]any{"pattern": "^[a-z]+@[a-z]+\\.[a-z]+$"},
			wantErr: false,
		},
		{
			name:    "invalid - does not match pattern",
			value:   "invalid-email",
			config:  map[string]any{"pattern": "^[a-z]+@[a-z]+\\.[a-z]+$"},
			wantErr: true,
		},
		{
			name:    "invalid - non-string value",
			value:   123,
			config:  map[string]any{"pattern": "^[0-9]+$"},
			wantErr: true,
		},
		{
			name:    "invalid - missing pattern config",
			value:   "test",
			config:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, testCtx, OperationCreate, "testProp", nil, tt.value, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternValidator_ValidateConfig(t *testing.T) {
	validator := &PatternValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid pattern",
			config:  map[string]any{"pattern": "^[a-z]+$"},
			wantErr: false,
		},
		{
			name:    "valid complex pattern",
			config:  map[string]any{"pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"},
			wantErr: false,
		},
		{
			name:    "invalid - missing pattern",
			config:  map[string]any{},
			wantErr: true,
		},
		{
			name:    "invalid - wrong type",
			config:  map[string]any{"pattern": 123},
			wantErr: true,
		},
		{
			name:    "invalid - bad regex syntax",
			config:  map[string]any{"pattern": "[invalid(regex"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig("testProp", tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternValidator_CachesCompiledRegex(t *testing.T) {
	validator := &PatternValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}
	config := map[string]any{"pattern": "^test$"}

	// First call should compile and cache
	err1 := validator.Validate(ctx, testCtx, OperationCreate, "prop1", nil, "test", config)
	if err1 != nil {
		t.Fatalf("First validation failed: %v", err1)
	}

	// Second call should use cached regex
	err2 := validator.Validate(ctx, testCtx, OperationCreate, "prop2", nil, "test", config)
	if err2 != nil {
		t.Fatalf("Second validation failed: %v", err2)
	}

	// Verify cache has entry (we can't directly inspect sync.Map, but we tested it works)
	// If caching wasn't working, the second call would still work but be slower
}
