package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewAgentType(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		wantErr  bool
	}{
		{
			name:     "Valid agent type",
			typeName: "VM Runner",
			wantErr:  false,
		},
		{
			name:     "Empty name",
			typeName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentType, err := NewAgentType(tt.typeName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAgentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if agentType.Name != tt.typeName {
				t.Errorf("NewAgentType() name = %v, want %v", agentType.Name, tt.typeName)
			}

			if len(agentType.ServiceTypes) != 0 {
				t.Errorf("NewAgentType() serviceTypes length = %v, want 0", len(agentType.ServiceTypes))
			}
		})
	}
}

func TestAgentTypeValidate(t *testing.T) {
	tests := []struct {
		name      string
		agentType *AgentType
		wantErr   bool
	}{
		{
			name: "Valid agent type",
			agentType: &AgentType{
				Name: "VM Runner",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			agentType: &AgentType{
				Name: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.agentType.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("AgentType.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentTypeServiceTypeManagement(t *testing.T) {
	agentType := &AgentType{
		BaseEntity: BaseEntity{
			ID: uuid.New(),
		},
		Name: "VM Runner",
	}

	serviceType1 := &ServiceType{
		BaseEntity: BaseEntity{
			ID: uuid.New(),
		},
		Name: "VM Service",
	}

	serviceType2 := &ServiceType{
		BaseEntity: BaseEntity{
			ID: uuid.New(),
		},
		Name: "Container Service",
	}

	t.Run("Add service type", func(t *testing.T) {
		if err := agentType.AddServiceType(serviceType1); err != nil {
			t.Errorf("AddServiceType() error = %v", err)
		}
		if len(agentType.ServiceTypes) != 1 {
			t.Errorf("AddServiceType() serviceTypes length = %v, want 1", len(agentType.ServiceTypes))
		}
	})

	t.Run("Add second service type", func(t *testing.T) {
		if err := agentType.AddServiceType(serviceType2); err != nil {
			t.Errorf("AddServiceType() error = %v", err)
		}
		if len(agentType.ServiceTypes) != 2 {
			t.Errorf("AddServiceType() serviceTypes length = %v, want 2", len(agentType.ServiceTypes))
		}
	})

	t.Run("Add nil service type", func(t *testing.T) {
		if err := agentType.AddServiceType(nil); err == nil {
			t.Error("AddServiceType() expected error for nil service type")
		}
	})

	t.Run("Has service type", func(t *testing.T) {
		if !agentType.HasServiceType(serviceType1.ID) {
			t.Error("HasServiceType() returned false for existing service type")
		}
		if !agentType.HasServiceType(serviceType2.ID) {
			t.Error("HasServiceType() returned false for existing service type")
		}
		if agentType.HasServiceType(uuid.New()) {
			t.Error("HasServiceType() returned true for non-existent service type")
		}
	})

	t.Run("Remove service type", func(t *testing.T) {
		if err := agentType.RemoveServiceType(serviceType1.ID); err != nil {
			t.Errorf("RemoveServiceType() error = %v", err)
		}
		if len(agentType.ServiceTypes) != 1 {
			t.Errorf("RemoveServiceType() serviceTypes length = %v, want 1", len(agentType.ServiceTypes))
		}
		if agentType.HasServiceType(serviceType1.ID) {
			t.Error("RemoveServiceType() service type still exists after removal")
		}
	})

	t.Run("Remove non-existent service type", func(t *testing.T) {
		if err := agentType.RemoveServiceType(uuid.New()); err == nil {
			t.Error("RemoveServiceType() expected error for non-existent service type")
		}
	})
}
