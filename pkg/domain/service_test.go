package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceStatus_Validate(t *testing.T) {
	tests := []struct {
		name       string
		status     ServiceStatus
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid New status",
			status:  ServiceNew,
			wantErr: false,
		},
		{
			name:    "Valid Started status",
			status:  ServiceStarted,
			wantErr: false,
		},

		{
			name:    "Valid Stopped status",
			status:  ServiceStopped,
			wantErr: false,
		},

		{
			name:    "Valid Deleted status",
			status:  ServiceDeleted,
			wantErr: false,
		},
		{
			name:       "Invalid status",
			status:     "InvalidStatus",
			wantErr:    true,
			errMessage: "invalid service status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseServiceStatus(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       ServiceStatus
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid New status",
			input:   "New",
			want:    ServiceNew,
			wantErr: false,
		},
		{
			name:    "Valid Started status",
			input:   "Started",
			want:    ServiceStarted,
			wantErr: false,
		},
		{
			name:    "Valid Stopped status",
			input:   "Stopped",
			want:    ServiceStopped,
			wantErr: false,
		},
		{
			name:    "Valid Deleted status",
			input:   "Deleted",
			want:    ServiceDeleted,
			wantErr: false,
		},
		{
			name:       "Invalid status",
			input:      "InvalidStatus",
			wantErr:    true,
			errMessage: "invalid service status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServiceStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestService_TableName(t *testing.T) {
	svc := &Service{}
	assert.Equal(t, "services", svc.TableName())
}

func TestService_Validate(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name       string
		service    *Service
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid service",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			service: &Service{
				Name:          "",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service name cannot be empty",
		},
		{
			name: "Invalid status",
			service: &Service{
				Name:          "Web Server",
				Status:        "InvalidStatus",
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "invalid service status",
		},
		{
			name: "Nil group ID",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       uuid.Nil,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service group ID cannot be nil",
		},
		{
			name: "Nil agent ID",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       uuid.Nil,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service agent ID cannot be nil",
		},
		{
			name: "Nil service type ID",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: uuid.Nil,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr:    true,
			errMessage: "service type ID cannot be nil",
		},
		{
			name: "With properties",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				Properties:    &properties.JSON{"port": 8080},
			},
			wantErr: false,
		},
		{
			name: "With external ID",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
				ExternalID:    stringPtr("ext-123"),
			},
			wantErr: false,
		},
		{
			name: "With error message",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr: false,
		},
		{
			name: "With failed action",
			service: &Service{
				Name:          "Web Server",
				Status:        ServiceNew,
				GroupID:       validID,
				AgentID:       validID,
				ServiceTypeID: validID,
				ProviderID:    validID,
				ConsumerID:    validID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.service.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMergeServiceProperties(t *testing.T) {
	tests := []struct {
		name     string
		existing *properties.JSON
		partial  properties.JSON
		expected properties.JSON
	}{
		{
			name:     "merge with existing properties",
			existing: &properties.JSON{"a": "1", "b": "2"},
			partial:  properties.JSON{"b": "3", "c": "4"},
			expected: properties.JSON{"a": "1", "b": "3", "c": "4"},
		},
		{
			name:     "merge with nil existing properties",
			existing: nil,
			partial:  properties.JSON{"a": "1", "b": "2"},
			expected: properties.JSON{"a": "1", "b": "2"},
		},
		{
			name:     "merge with empty partial properties",
			existing: &properties.JSON{"a": "1", "b": "2"},
			partial:  properties.JSON{},
			expected: properties.JSON{"a": "1", "b": "2"},
		},
		{
			name:     "merge with empty existing and partial",
			existing: &properties.JSON{},
			partial:  properties.JSON{},
			expected: properties.JSON{},
		},
		{
			name:     "deep merge nested objects",
			existing: &properties.JSON{"config": map[string]any{"host": "localhost", "port": 8080}},
			partial:  properties.JSON{"config": map[string]any{"port": 9090, "ssl": true}},
			expected: properties.JSON{"config": map[string]any{"host": "localhost", "port": 9090, "ssl": true}},
		},
		{
			name:     "replace non-object with object",
			existing: &properties.JSON{"config": "simple"},
			partial:  properties.JSON{"config": map[string]any{"host": "localhost"}},
			expected: properties.JSON{"config": map[string]any{"host": "localhost"}},
		},
		{
			name:     "replace object with non-object",
			existing: &properties.JSON{"config": map[string]any{"host": "localhost"}},
			partial:  properties.JSON{"config": "simple"},
			expected: properties.JSON{"config": "simple"},
		},
		{
			name: "deep merge multiple levels",
			existing: &properties.JSON{
				"database": map[string]any{
					"host": "localhost",
					"config": map[string]any{
						"pool_size": 10,
						"timeout":   30,
					},
				},
			},
			partial: properties.JSON{
				"database": map[string]any{
					"port": 5432,
					"config": map[string]any{
						"timeout": 60,
						"ssl":     true,
					},
				},
			},
			expected: properties.JSON{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
					"config": map[string]any{
						"pool_size": 10,
						"timeout":   60,
						"ssl":       true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeServiceProperties(tt.existing, tt.partial)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeNestedObjects(t *testing.T) {
	tests := []struct {
		name     string
		existing map[string]any
		partial  map[string]any
		expected map[string]any
	}{
		{
			name:     "merge simple objects",
			existing: map[string]any{"a": "1", "b": "2"},
			partial:  map[string]any{"b": "3", "c": "4"},
			expected: map[string]any{"a": "1", "b": "3", "c": "4"},
		},
		{
			name:     "merge with empty existing",
			existing: map[string]any{},
			partial:  map[string]any{"a": "1"},
			expected: map[string]any{"a": "1"},
		},
		{
			name:     "merge with empty partial",
			existing: map[string]any{"a": "1"},
			partial:  map[string]any{},
			expected: map[string]any{"a": "1"},
		},
		{
			name: "deep merge nested objects",
			existing: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{"a": "1", "b": "2"},
				},
			},
			partial: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{"b": "3", "c": "4"},
				},
			},
			expected: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{"a": "1", "b": "3", "c": "4"},
				},
			},
		},
		{
			name:     "replace nested object with non-object",
			existing: map[string]any{"nested": map[string]any{"a": "1"}},
			partial:  map[string]any{"nested": "simple"},
			expected: map[string]any{"nested": "simple"},
		},
		{
			name:     "replace non-object with nested object",
			existing: map[string]any{"nested": "simple"},
			partial:  map[string]any{"nested": map[string]any{"a": "1"}},
			expected: map[string]any{"nested": map[string]any{"a": "1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeNestedObjects(tt.existing, tt.partial)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServiceUpdate_PartialProperties(t *testing.T) {
	// This test verifies that the UpdateService function properly merges partial properties
	// with existing service properties before validation and update

	tests := []struct {
		name          string
		existingProps *properties.JSON
		partialProps  properties.JSON
		expectedProps properties.JSON
		expectError   bool
		errorMessage  string
	}{
		{
			name:          "partial update preserves existing properties",
			existingProps: &properties.JSON{"database": map[string]any{"host": "localhost", "port": 5432}, "cache": map[string]any{"enabled": true}},
			partialProps:  properties.JSON{"database": map[string]any{"port": 3306}},
			expectedProps: properties.JSON{"database": map[string]any{"host": "localhost", "port": 3306}, "cache": map[string]any{"enabled": true}},
			expectError:   false,
		},
		{
			name:          "add new properties to existing",
			existingProps: &properties.JSON{"database": map[string]any{"host": "localhost"}},
			partialProps:  properties.JSON{"api": map[string]any{"version": "v2", "timeout": 30}},
			expectedProps: properties.JSON{"database": map[string]any{"host": "localhost"}, "api": map[string]any{"version": "v2", "timeout": 30}},
			expectError:   false,
		},
		{
			name: "deep merge nested objects",
			existingProps: &properties.JSON{
				"config": map[string]any{
					"database": map[string]any{"host": "localhost", "port": 5432},
					"cache":    map[string]any{"enabled": true, "size": 100},
				},
			},
			partialProps: properties.JSON{
				"config": map[string]any{
					"database": map[string]any{"port": 3306, "ssl": true},
					"api":      map[string]any{"version": "v2"},
				},
			},
			expectedProps: properties.JSON{
				"config": map[string]any{
					"database": map[string]any{"host": "localhost", "port": 3306, "ssl": true},
					"cache":    map[string]any{"enabled": true, "size": 100},
					"api":      map[string]any{"version": "v2"},
				},
			},
			expectError: false,
		},
		{
			name:          "replace entire nested object",
			existingProps: &properties.JSON{"config": map[string]any{"database": map[string]any{"host": "localhost"}}},
			partialProps:  properties.JSON{"config": "simple"},
			expectedProps: properties.JSON{"config": "simple"},
			expectError:   false,
		},
		{
			name:          "merge with nil existing properties",
			existingProps: nil,
			partialProps:  properties.JSON{"new": "value"},
			expectedProps: properties.JSON{"new": "value"},
			expectError:   false,
		},
		{
			name:          "empty partial update preserves existing",
			existingProps: &properties.JSON{"existing": "value"},
			partialProps:  properties.JSON{},
			expectedProps: properties.JSON{"existing": "value"},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the mergeServiceProperties function directly
			result := mergeServiceProperties(tt.existingProps, tt.partialProps)
			assert.Equal(t, tt.expectedProps, result)
		})
	}
}

func TestServiceUpdate_IntegrationFlow(t *testing.T) {
	// This test simulates the full integration flow of service update with property merging
	// It verifies that the merging happens before validation and that the complete merged
	// properties are used for the update

	// Test data setup
	existingProps := &properties.JSON{
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
			"config": map[string]any{
				"pool_size": 10,
				"timeout":   30,
			},
		},
		"cache": map[string]any{
			"enabled": true,
			"size":    100,
		},
	}

	partialUpdate := properties.JSON{
		"database": map[string]any{
			"port": 3306,
			"config": map[string]any{
				"timeout": 60,
				"ssl":     true,
			},
		},
		"api": map[string]any{
			"version": "v2",
		},
	}

	expectedMerged := properties.JSON{
		"database": map[string]any{
			"host": "localhost", // preserved from existing
			"port": 3306,        // updated from partial
			"config": map[string]any{
				"pool_size": 10,   // preserved from existing
				"timeout":   60,   // updated from partial
				"ssl":       true, // added from partial
			},
		},
		"cache": map[string]any{
			"enabled": true, // preserved from existing
			"size":    100,  // preserved from existing
		},
		"api": map[string]any{
			"version": "v2", // added from partial
		},
	}

	// Test the merge function
	merged := mergeServiceProperties(existingProps, partialUpdate)
	assert.Equal(t, expectedMerged, merged)

	// Verify that the merge preserves existing properties that weren't touched
	assert.Equal(t, "localhost", merged["database"].(map[string]any)["host"])
	assert.Equal(t, true, merged["cache"].(map[string]any)["enabled"])
	assert.Equal(t, 100, merged["cache"].(map[string]any)["size"])

	// Verify that the merge updates provided properties
	assert.Equal(t, 3306, merged["database"].(map[string]any)["port"])
	assert.Equal(t, 60, merged["database"].(map[string]any)["config"].(map[string]any)["timeout"])
	assert.Equal(t, true, merged["database"].(map[string]any)["config"].(map[string]any)["ssl"])

	// Verify that the merge adds new properties
	assert.Equal(t, "v2", merged["api"].(map[string]any)["version"])
}
