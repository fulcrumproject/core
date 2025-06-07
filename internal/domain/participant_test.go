package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestParticipantStatus_Validate(t *testing.T) {
	tests := []struct {
		name    string
		status  ParticipantStatus
		wantErr bool
	}{
		{
			name:    "Enabled status",
			status:  ParticipantEnabled,
			wantErr: false,
		},
		{
			name:    "Disabled status",
			status:  ParticipantDisabled,
			wantErr: false,
		},
		{
			name:    "Invalid status",
			status:  "InvalidStatus",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseParticipantStatus(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    ParticipantStatus
		wantErr bool
	}{
		{
			name:    "Parse Enabled status",
			value:   "Enabled",
			want:    ParticipantEnabled,
			wantErr: false,
		},
		{
			name:    "Parse Disabled status",
			value:   "Disabled",
			want:    ParticipantDisabled,
			wantErr: false,
		},
		{
			name:    "Parse invalid status",
			value:   "InvalidStatus",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParticipantStatus(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParticipant_TableName(t *testing.T) {
	participant := Participant{}
	assert.Equal(t, "participants", participant.TableName())
}

func TestParticipant_Validate(t *testing.T) {
	validID := uuid.New()
	_ = validID // prevent unused error for now, will be used later

	tests := []struct {
		name        string
		participant *Participant
		wantErr     bool
		errContains string
	}{
		{
			name: "Valid participant",
			participant: &Participant{
				Name:   "test-participant",
				Status: ParticipantEnabled,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			participant: &Participant{
				Name:   "",
				Status: ParticipantEnabled,
			},
			wantErr:     true,
			errContains: "participant name cannot be empty",
		},
		{
			name: "Invalid status",
			participant: &Participant{
				Name:   "test-participant",
				Status: "InvalidStatus",
			},
			wantErr:     true,
			errContains: "invalid participant status",
		},
		{
			name: "Valid with empty country code",
			participant: &Participant{
				Name:   "test-participant",
				Status: ParticipantEnabled,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.participant.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
