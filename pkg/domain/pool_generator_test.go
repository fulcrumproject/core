package domain

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRetention(t *testing.T) {
	tests := []struct {
		name    string
		cfg     properties.JSON
		want    time.Duration
		wantErr bool
	}{
		{name: "absent → immediate", cfg: properties.JSON{}, want: 0},
		{name: "zero → immediate", cfg: properties.JSON{"retentionSeconds": 0}, want: 0},
		{name: "positive seconds", cfg: properties.JSON{"retentionSeconds": 3600}, want: time.Hour},
		{name: "float64 (JSON number) seconds", cfg: properties.JSON{"retentionSeconds": float64(90)}, want: 90 * time.Second},
		{name: "negative rejected", cfg: properties.JSON{"retentionSeconds": -1}, wantErr: true},
		{name: "string rejected", cfg: properties.JSON{"retentionSeconds": "3600"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRetention(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseNeverReallocate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     properties.JSON
		want    bool
		wantErr bool
	}{
		{name: "absent → false", cfg: properties.JSON{}, want: false},
		{name: "true", cfg: properties.JSON{"neverReallocate": true}, want: true},
		{name: "false", cfg: properties.JSON{"neverReallocate": false}, want: false},
		{name: "string rejected", cfg: properties.JSON{"neverReallocate": "true"}, wantErr: true},
		{name: "number rejected", cfg: properties.JSON{"neverReallocate": 1}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNeverReallocate(tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRetentionAllows(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)

	tests := []struct {
		name            string
		retention       time.Duration
		neverReallocate bool
		releasedAt      *time.Time
		want            bool
	}{
		{name: "never released → always allowed", retention: time.Hour, releasedAt: nil, want: true},
		{name: "immediate cooldown → allowed right after release", retention: 0, releasedAt: &now, want: true},
		{name: "within cooldown → blocked", retention: 2 * time.Hour, releasedAt: &past, want: false},
		{name: "cooldown elapsed → allowed", retention: 30 * time.Minute, releasedAt: &past, want: true},
		{name: "exactly at boundary → allowed", retention: time.Hour, releasedAt: &past, want: true},
		{name: "neverReallocate + released → blocked", neverReallocate: true, releasedAt: &past, want: false},
		{name: "neverReallocate + never released → allowed", neverReallocate: true, releasedAt: nil, want: true},
		{name: "neverReallocate wins over elapsed cooldown → blocked", retention: 30 * time.Minute, neverReallocate: true, releasedAt: &past, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, retentionAllows(tt.retention, tt.neverReallocate, tt.releasedAt, now))
		})
	}
}
