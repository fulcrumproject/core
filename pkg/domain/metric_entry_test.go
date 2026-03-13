package domain

import (
	"testing"
	"time"

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

func TestAggregateBucket_MaxDuration(t *testing.T) {
	assert.Equal(t, 24*time.Hour, AggregateBucketMinute.MaxDuration())
	assert.Equal(t, 7*24*time.Hour, AggregateBucketHour.MaxDuration())
	assert.Equal(t, 90*24*time.Hour, AggregateBucketDay.MaxDuration())
	assert.Equal(t, 365*24*time.Hour, AggregateBucketMonth.MaxDuration())
}

func TestAggregateBucket_ValidateTimeRange(t *testing.T) {
	end := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	t.Run("Valid range for minute bucket", func(t *testing.T) {
		start := end.Add(-12 * time.Hour)
		assert.NoError(t, AggregateBucketMinute.ValidateTimeRange(start, end))
	})

	t.Run("Exceeds max for minute bucket", func(t *testing.T) {
		start := end.Add(-48 * time.Hour)
		assert.Error(t, AggregateBucketMinute.ValidateTimeRange(start, end))
	})

	t.Run("End before start", func(t *testing.T) {
		start := end.Add(1 * time.Hour)
		assert.Error(t, AggregateBucketMinute.ValidateTimeRange(start, end))
	})

	t.Run("Exact max duration is valid", func(t *testing.T) {
		start := end.Add(-24 * time.Hour)
		assert.NoError(t, AggregateBucketMinute.ValidateTimeRange(start, end))
	})
}

func TestAggregateBucket_DefaultStart(t *testing.T) {
	end := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, end.Add(-24*time.Hour), AggregateBucketMinute.DefaultStart(end))
	assert.Equal(t, end.Add(-7*24*time.Hour), AggregateBucketHour.DefaultStart(end))
}
