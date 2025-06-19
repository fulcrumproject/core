package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMetricEntry_TableName(t *testing.T) {
	entry := MetricEntry{}
	assert.Equal(t, "metric_entries", entry.TableName())
}

func TestMetricEntry_Validate(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name       string
		entry      *MetricEntry
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid entry",
			entry: &MetricEntry{
				ResourceID: "cpu",
				Value:      42.5,
				TypeID:     validID,
				AgentID:    validID,
				ServiceID:  validID,
				ProviderID: validID,
				ConsumerID: validID,
			},
			wantErr: false,
		},
		{
			name: "Empty ResourceID",
			entry: &MetricEntry{
				ResourceID: "",
				Value:      42.5,
				TypeID:     validID,
				AgentID:    validID,
				ServiceID:  validID,
				ProviderID: validID,
				ConsumerID: validID,
			},
			wantErr:    true,
			errMessage: "resource ID cannot be empty",
		},
		{
			name: "Empty TypeID",
			entry: &MetricEntry{
				ResourceID: "cpu",
				Value:      42.5,
				TypeID:     uuid.Nil,
				AgentID:    validID,
				ServiceID:  validID,
				ProviderID: validID,
				ConsumerID: validID,
			},
			wantErr:    true,
			errMessage: "metric type ID cannot be empty",
		},
		{
			name: "Empty AgentID",
			entry: &MetricEntry{
				ResourceID: "cpu",
				Value:      42.5,
				TypeID:     validID,
				AgentID:    uuid.Nil,
				ServiceID:  validID,
				ProviderID: validID,
				ConsumerID: validID,
			},
			wantErr:    true,
			errMessage: "agent ID cannot be empty",
		},
		{
			name: "Empty ServiceID",
			entry: &MetricEntry{
				ResourceID: "cpu",
				Value:      42.5,
				TypeID:     validID,
				AgentID:    validID,
				ServiceID:  uuid.Nil,
				ProviderID: validID,
				ConsumerID: validID,
			},
			wantErr:    true,
			errMessage: "service ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
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
