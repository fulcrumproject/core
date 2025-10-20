// Tests for ServiceOptionType entity
package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceOptionType_TableName(t *testing.T) {
	sot := ServiceOptionType{}
	assert.Equal(t, "service_option_types", sot.TableName())
}

func TestServiceOptionType_Validate(t *testing.T) {
	tests := []struct {
		name       string
		optionType *ServiceOptionType
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid service option type",
			optionType: &ServiceOptionType{
				Name: "VM Images",
				Type: "vmImages",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			optionType: &ServiceOptionType{
				Name: "",
				Type: "vmImages",
			},
			wantErr:    true,
			errMessage: "name cannot be empty",
		},
		{
			name: "Empty type",
			optionType: &ServiceOptionType{
				Name: "VM Images",
				Type: "",
			},
			wantErr:    true,
			errMessage: "type cannot be empty",
		},
		{
			name: "Valid type with underscores",
			optionType: &ServiceOptionType{
				Name: "Machine Types",
				Type: "machine_types_v2",
			},
			wantErr: false,
		},
		{
			name: "Valid type with numbers",
			optionType: &ServiceOptionType{
				Name: "Machine Types",
				Type: "machineTypes2024",
			},
			wantErr: false,
		},
		{
			name: "Invalid type with spaces",
			optionType: &ServiceOptionType{
				Name: "VM Images",
				Type: "vm images",
			},
			wantErr:    true,
			errMessage: "type must contain only alphanumeric characters and underscores",
		},
		{
			name: "Invalid type with dashes",
			optionType: &ServiceOptionType{
				Name: "VM Images",
				Type: "vm-images",
			},
			wantErr:    true,
			errMessage: "type must contain only alphanumeric characters and underscores",
		},
		{
			name: "Invalid type with special characters",
			optionType: &ServiceOptionType{
				Name: "VM Images",
				Type: "vm@images",
			},
			wantErr:    true,
			errMessage: "type must contain only alphanumeric characters and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.optionType.Validate()
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

func TestNewServiceOptionType(t *testing.T) {
	params := CreateServiceOptionTypeParams{
		Name:        "VM Images",
		Type:        "vmImages",
		Description: "Available VM images",
	}

	optionType := NewServiceOptionType(params)

	assert.NotNil(t, optionType)
	assert.Equal(t, "VM Images", optionType.Name)
	assert.Equal(t, "vmImages", optionType.Type)
	assert.Equal(t, "Available VM images", optionType.Description)
}

func TestServiceOptionType_Update(t *testing.T) {
	optionType := &ServiceOptionType{
		Name:        "VM Images",
		Type:        "vmImages",
		Description: "Old description",
	}

	newName := "Virtual Machine Images"
	newDesc := "New description"

	optionType.Update(UpdateServiceOptionTypeParams{
		Name:        &newName,
		Description: &newDesc,
	})

	assert.Equal(t, "Virtual Machine Images", optionType.Name)
	assert.Equal(t, "New description", optionType.Description)
	// Type should not change
	assert.Equal(t, "vmImages", optionType.Type)
}

func TestServiceOptionType_Update_Partial(t *testing.T) {
	optionType := &ServiceOptionType{
		Name:        "VM Images",
		Type:        "vmImages",
		Description: "Old description",
	}

	newName := "Virtual Machine Images"

	optionType.Update(UpdateServiceOptionTypeParams{
		Name: &newName,
		// Description is nil, should not update
	})

	assert.Equal(t, "Virtual Machine Images", optionType.Name)
	assert.Equal(t, "Old description", optionType.Description)
}

