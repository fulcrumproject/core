// Tests for ServiceOption entity
package domain

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceOption_TableName(t *testing.T) {
	so := ServiceOption{}
	assert.Equal(t, "service_options", so.TableName())
}

func TestServiceOption_Validate(t *testing.T) {
	providerID := properties.UUID(uuid.New())
	optionTypeID := properties.UUID(uuid.New())

	tests := []struct {
		name       string
		option     *ServiceOption
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid service option",
			option: &ServiceOption{
				ProviderID:          providerID,
				ServiceOptionTypeID: optionTypeID,
				Name:                "Ubuntu 22.04",
				Value:               map[string]any{"image": "ubuntu-22.04"},
				Enabled:             true,
			},
			wantErr: false,
		},
		{
			name: "Valid service option with string value",
			option: &ServiceOption{
				ProviderID:          providerID,
				ServiceOptionTypeID: optionTypeID,
				Name:                "Ubuntu 22.04",
				Value:               "ubuntu-22.04",
				Enabled:             true,
			},
			wantErr: false,
		},
		{
			name: "Valid disabled option",
			option: &ServiceOption{
				ProviderID:          providerID,
				ServiceOptionTypeID: optionTypeID,
				Name:                "Ubuntu 20.04",
				Value:               "ubuntu-20.04",
				Enabled:             false,
			},
			wantErr: false,
		},
		{
			name: "Empty provider ID",
			option: &ServiceOption{
				ProviderID:          properties.UUID(uuid.Nil),
				ServiceOptionTypeID: optionTypeID,
				Name:                "Ubuntu 22.04",
				Value:               "ubuntu-22.04",
				Enabled:             true,
			},
			wantErr:    true,
			errMessage: "providerId cannot be empty",
		},
		{
			name: "Empty option type ID",
			option: &ServiceOption{
				ProviderID:          providerID,
				ServiceOptionTypeID: properties.UUID(uuid.Nil),
				Name:                "Ubuntu 22.04",
				Value:               "ubuntu-22.04",
				Enabled:             true,
			},
			wantErr:    true,
			errMessage: "serviceOptionTypeId cannot be empty",
		},
		{
			name: "Empty name",
			option: &ServiceOption{
				ProviderID:          providerID,
				ServiceOptionTypeID: optionTypeID,
				Name:                "",
				Value:               "ubuntu-22.04",
				Enabled:             true,
			},
			wantErr:    true,
			errMessage: "name cannot be empty",
		},
		{
			name: "Nil value",
			option: &ServiceOption{
				ProviderID:          providerID,
				ServiceOptionTypeID: optionTypeID,
				Name:                "Ubuntu 22.04",
				Value:               nil,
				Enabled:             true,
			},
			wantErr:    true,
			errMessage: "value cannot be nil",
		},
		{
			name: "Valid with display order",
			option: &ServiceOption{
				ProviderID:          providerID,
				ServiceOptionTypeID: optionTypeID,
				Name:                "Ubuntu 22.04",
				Value:               "ubuntu-22.04",
				Enabled:             true,
				DisplayOrder:        10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.option.Validate()
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

func TestNewServiceOption(t *testing.T) {
	providerID := properties.UUID(uuid.New())
	optionTypeID := properties.UUID(uuid.New())

	params := CreateServiceOptionParams{
		ProviderID:          providerID,
		ServiceOptionTypeID: optionTypeID,
		Name:                "Ubuntu 22.04",
		Value:               map[string]any{"image": "ubuntu-22.04"},
		Enabled:             true,
		DisplayOrder:        5,
	}

	option := NewServiceOption(params)

	assert.NotNil(t, option)
	assert.Equal(t, providerID, option.ProviderID)
	assert.Equal(t, optionTypeID, option.ServiceOptionTypeID)
	assert.Equal(t, "Ubuntu 22.04", option.Name)
	assert.Equal(t, map[string]any{"image": "ubuntu-22.04"}, option.Value)
	assert.True(t, option.Enabled)
	assert.Equal(t, 5, option.DisplayOrder)
}

func TestServiceOption_Update(t *testing.T) {
	providerID := properties.UUID(uuid.New())
	optionTypeID := properties.UUID(uuid.New())

	option := &ServiceOption{
		ProviderID:          providerID,
		ServiceOptionTypeID: optionTypeID,
		Name:                "Ubuntu 22.04",
		Value:               "ubuntu-22.04",
		Enabled:             true,
		DisplayOrder:        0,
	}

	newName := "Ubuntu 22.04 LTS"
	newValue := any(map[string]any{"image": "ubuntu-22.04-lts"})
	newEnabled := false
	newDisplayOrder := 10

	option.Update(UpdateServiceOptionParams{
		Name:         &newName,
		Value:        &newValue,
		Enabled:      &newEnabled,
		DisplayOrder: &newDisplayOrder,
	})

	assert.Equal(t, "Ubuntu 22.04 LTS", option.Name)
	assert.Equal(t, map[string]any{"image": "ubuntu-22.04-lts"}, option.Value)
	assert.False(t, option.Enabled)
	assert.Equal(t, 10, option.DisplayOrder)

	// IDs should not change
	assert.Equal(t, providerID, option.ProviderID)
	assert.Equal(t, optionTypeID, option.ServiceOptionTypeID)
}

func TestServiceOption_Update_Partial(t *testing.T) {
	providerID := properties.UUID(uuid.New())
	optionTypeID := properties.UUID(uuid.New())

	option := &ServiceOption{
		ProviderID:          providerID,
		ServiceOptionTypeID: optionTypeID,
		Name:                "Ubuntu 22.04",
		Value:               "ubuntu-22.04",
		Enabled:             true,
		DisplayOrder:        5,
	}

	newEnabled := false

	option.Update(UpdateServiceOptionParams{
		Enabled: &newEnabled,
		// Other fields are nil, should not update
	})

	assert.Equal(t, "Ubuntu 22.04", option.Name)
	assert.Equal(t, "ubuntu-22.04", option.Value)
	assert.False(t, option.Enabled)
	assert.Equal(t, 5, option.DisplayOrder)
}

