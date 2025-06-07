package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceGroup_Validate(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name       string
		sg         *ServiceGroup
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid service group",
			sg: &ServiceGroup{
				Name:       "Test Group",
				ConsumerID: validID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			sg: &ServiceGroup{
				Name:       "",
				ConsumerID: validID,
			},
			wantErr:    true,
			errMessage: "service group name cannot be empty",
		},
		{
			name: "Nil consumer ID",
			sg: &ServiceGroup{
				Name:       "Test Group",
				ConsumerID: uuid.Nil,
			},
			wantErr:    true,
			errMessage: "service group consumer cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sg.Validate()
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

func TestServiceGroup_TableName(t *testing.T) {
	sg := ServiceGroup{}
	assert.Equal(t, "service_groups", sg.TableName())
}
