package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestToken_TableName(t *testing.T) {
	token := Token{}
	assert.Equal(t, "tokens", token.TableName())
}

func TestToken_Validate(t *testing.T) {
	validID := uuid.New()
	now := time.Now().Add(24 * time.Hour) // 1 day in the future

	tests := []struct {
		name       string
		token      *Token
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid fulcrum admin token",
			token: &Token{
				Name:        "Admin Token",
				Role:        RoleFulcrumAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr: false,
		},
		{
			name: "Valid participant token",
			token: &Token{
				Name:          "Provider Admin Token",
				Role:          RoleParticipant,
				HashedValue:   "hashedvalue",
				ExpireAt:      now,
				ParticipantID: &validID,
			},
			wantErr: false,
		},
		{
			name: "Valid agent token",
			token: &Token{
				Name:          "Agent Token",
				Role:          RoleAgent,
				HashedValue:   "hashedvalue",
				ExpireAt:      now,
				AgentID:       &validID,
				ParticipantID: &validID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			token: &Token{
				Name:        "",
				Role:        RoleFulcrumAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "token name cannot be empty",
		},
		{
			name: "Empty hashed value",
			token: &Token{
				Name:        "Admin Token",
				Role:        RoleFulcrumAdmin,
				HashedValue: "",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "token hashed value cannot be empty",
		},
		{
			name: "Invalid role",
			token: &Token{
				Name:        "Admin Token",
				Role:        "invalid-role",
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "invalid auth role",
		},
		{
			name: "Zero expire time",
			token: &Token{
				Name:        "Admin Token",
				Role:        RoleFulcrumAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    time.Time{},
			},
			wantErr:    true,
			errMessage: "token expire at cannot be empty",
		},
		{
			name: "Fulcrum admin with participant ID",
			token: &Token{
				Name:          "Admin Token",
				Role:          RoleFulcrumAdmin,
				HashedValue:   "hashedvalue",
				ExpireAt:      now,
				ParticipantID: &validID,
			},
			wantErr:    true,
			errMessage: "fulcrum admin tokens should not have any scope IDs",
		},
		{
			name: "Provider admin without participant ID",
			token: &Token{
				Name:        "Provider Admin Token",
				Role:        RoleParticipant,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "participant ID is required for participant role",
		},
		{
			name: "Agent without agent ID",
			token: &Token{
				Name:          "Agent Token",
				Role:          RoleAgent,
				HashedValue:   "hashedvalue",
				ExpireAt:      now,
				ParticipantID: &validID,
			},
			wantErr:    true,
			errMessage: "agent ID is required for agent role",
		},
		{
			name: "Agent without participant ID",
			token: &Token{
				Name:        "Agent Token",
				Role:        RoleAgent,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				AgentID:     &validID,
			},
			wantErr:    true,
			errMessage: "participant ID is required for agent role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.token.Validate()
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

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		expireAt time.Time
		want     bool
	}{
		{
			name:     "Not expired",
			expireAt: time.Now().Add(24 * time.Hour), // 1 day in the future
			want:     false,
		},
		{
			name:     "Expired",
			expireAt: time.Now().Add(-24 * time.Hour), // 1 day in the past
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				ExpireAt: tt.expireAt,
			}
			got := token.IsExpired()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToken_GenerateTokenValue(t *testing.T) {
	token := &Token{}
	err := token.GenerateTokenValue()
	assert.NoError(t, err)
	assert.NotEmpty(t, token.PlainValue)
	assert.NotEmpty(t, token.HashedValue)
	assert.NotEqual(t, token.PlainValue, token.HashedValue)
}

func TestToken_VerifyTokenValue(t *testing.T) {
	token := &Token{}
	err := token.GenerateTokenValue()
	assert.NoError(t, err)

	validValue := token.PlainValue
	invalidValue := "invalid-token-value"

	assert.True(t, token.VerifyTokenValue(validValue))
	assert.False(t, token.VerifyTokenValue(invalidValue))
}

func TestHashTokenValue(t *testing.T) {
	value1 := "token1"
	value2 := "token2"

	hash1 := HashTokenValue(value1)
	hash2 := HashTokenValue(value2)
	hash1Again := HashTokenValue(value1)

	assert.NotEmpty(t, hash1)
	assert.NotEmpty(t, hash2)
	assert.NotEqual(t, hash1, hash2)
	assert.Equal(t, hash1, hash1Again)
}
