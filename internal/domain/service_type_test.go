package domain

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNewServiceType(t *testing.T) {
	tests := []struct {
		name                string
		typeName            string
		resourceDefinitions JSON
		wantErr             bool
	}{
		{
			name:     "Valid service type",
			typeName: "VM Service",
			resourceDefinitions: JSON{
				"cpu":    float64(4),
				"memory": "8GB",
				"disk": map[string]interface{}{
					"size": "100GB",
					"type": "SSD",
				},
			},
			wantErr: false,
		},
		{
			name:                "Empty name",
			typeName:            "",
			resourceDefinitions: JSON{},
			wantErr:             true,
		},
		{
			name:     "Invalid resource definitions",
			typeName: "VM Service",
			resourceDefinitions: JSON{
				"": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceType, err := NewServiceType(tt.typeName, tt.resourceDefinitions)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServiceType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if serviceType.Name != tt.typeName {
				t.Errorf("NewServiceType() name = %v, want %v", serviceType.Name, tt.typeName)
			}

			// Test resource definitions conversion
			defs, err := serviceType.GetResourceDefinitions()
			if err != nil {
				t.Errorf("GetResourceDefinitions() error = %v", err)
				return
			}

			// Deep comparison of JSON structures
			if !reflect.DeepEqual(defs, tt.resourceDefinitions) {
				defsJSON, _ := json.Marshal(defs)
				expectedJSON, _ := json.Marshal(tt.resourceDefinitions)
				t.Errorf("Resource definitions don't match\nGot: %s\nWant: %s", defsJSON, expectedJSON)
			}
		})
	}
}

func TestServiceTypeValidate(t *testing.T) {
	tests := []struct {
		name        string
		serviceType *ServiceType
		wantErr     bool
	}{
		{
			name: "Valid service type",
			serviceType: &ServiceType{
				Name: "VM Service",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			serviceType: &ServiceType{
				Name: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.serviceType.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ServiceType.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceTypeResourceDefinitions(t *testing.T) {
	initialDefs := JSON{
		"cpu":    float64(4),
		"memory": "8GB",
	}

	serviceType, err := NewServiceType("Test Service", initialDefs)
	if err != nil {
		t.Fatalf("Failed to create service type: %v", err)
	}

	t.Run("Get initial resource definitions", func(t *testing.T) {
		defs, err := serviceType.GetResourceDefinitions()
		if err != nil {
			t.Errorf("GetResourceDefinitions() error = %v", err)
			return
		}

		// Deep comparison of JSON structures
		if !reflect.DeepEqual(defs, initialDefs) {
			defsJSON, _ := json.Marshal(defs)
			initialDefsJSON, _ := json.Marshal(initialDefs)
			t.Errorf("Resource definitions don't match\nGot: %s\nWant: %s", defsJSON, initialDefsJSON)
		}
	})

	t.Run("Update resource definitions", func(t *testing.T) {
		newDefs := JSON{
			"cpu":    float64(8),
			"memory": "16GB",
			"disk":   "500GB",
		}

		if err := serviceType.UpdateResourceDefinitions(newDefs); err != nil {
			t.Errorf("UpdateResourceDefinitions() error = %v", err)
			return
		}

		defs, err := serviceType.GetResourceDefinitions()
		if err != nil {
			t.Errorf("GetResourceDefinitions() error = %v", err)
			return
		}

		// Deep comparison of JSON structures
		if !reflect.DeepEqual(defs, newDefs) {
			defsJSON, _ := json.Marshal(defs)
			newDefsJSON, _ := json.Marshal(newDefs)
			t.Errorf("Resource definitions don't match\nGot: %s\nWant: %s", defsJSON, newDefsJSON)
		}
	})

	t.Run("Update with invalid resource definitions", func(t *testing.T) {
		invalidDefs := JSON{
			"": "invalid",
		}

		if err := serviceType.UpdateResourceDefinitions(invalidDefs); err == nil {
			t.Error("UpdateResourceDefinitions() expected error for invalid definitions")
		}
	})
}
