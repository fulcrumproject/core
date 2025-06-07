package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricEntityType_Validate(t *testing.T) {
	tests := []struct {
		name       string
		entityType MetricEntityType
		wantErr    bool
		errMessage string
	}{
		{
			name:       "Valid MetricEntityTypeAgent",
			entityType: MetricEntityTypeAgent,
			wantErr:    false,
		},
		{
			name:       "Valid MetricEntityTypeService",
			entityType: MetricEntityTypeService,
			wantErr:    false,
		},
		{
			name:       "Valid MetricEntityTypeResource",
			entityType: MetricEntityTypeResource,
			wantErr:    false,
		},
		{
			name:       "Invalid entity type",
			entityType: "InvalidEntityType",
			wantErr:    true,
			errMessage: "invalid InvalidEntityType metric entity type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entityType.Validate()
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

func TestMetricType_TableName(t *testing.T) {
	metricType := MetricType{}
	assert.Equal(t, "metric_types", metricType.TableName())
}

func TestMetricType_Validate(t *testing.T) {
	tests := []struct {
		name       string
		metricType *MetricType
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid metric type",
			metricType: &MetricType{
				Name:       "cpu-usage",
				EntityType: MetricEntityTypeResource,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			metricType: &MetricType{
				Name:       "",
				EntityType: MetricEntityTypeResource,
			},
			wantErr:    true,
			errMessage: "metric type name cannot be empty",
		},
		{
			name: "Invalid entity type",
			metricType: &MetricType{
				Name:       "cpu-usage",
				EntityType: "InvalidEntityType",
			},
			wantErr:    true,
			errMessage: "invalid entity type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metricType.Validate()
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
