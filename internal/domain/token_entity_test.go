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
			name: "Valid provider admin token",
			token: &Token{
				Name:        "Provider Admin Token",
				Role:        RoleProviderAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				ProviderID:  &validID,
			},
			wantErr: false,
		},
		{
			name: "Valid broker token",
			token: &Token{
				Name:        "Broker Token",
				Role:        RoleBroker,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				BrokerID:    &validID,
			},
			wantErr: false,
		},
		{
			name: "Valid agent token",
			token: &Token{
				Name:        "Agent Token",
				Role:        RoleAgent,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				AgentID:     &validID,
				ProviderID:  &validID,
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
			name: "Fulcrum admin with provider ID",
			token: &Token{
				Name:        "Admin Token",
				Role:        RoleFulcrumAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				ProviderID:  &validID,
			},
			wantErr:    true,
			errMessage: "fulcrum admin tokens should not have any scope IDs",
		},
		{
			name: "Provider admin without provider ID",
			token: &Token{
				Name:        "Provider Admin Token",
				Role:        RoleProviderAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "provider ID is required for provider_admin role",
		},
		{
			name: "Provider admin with broker ID",
			token: &Token{
				Name:        "Provider Admin Token",
				Role:        RoleProviderAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				ProviderID:  &validID,
				BrokerID:    &validID,
			},
			wantErr:    true,
			errMessage: "provider_admin tokens should only have provider ID set",
		},
		{
			name: "Broker without broker ID",
			token: &Token{
				Name:        "Broker Token",
				Role:        RoleBroker,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "broker ID is required for broker role",
		},
		{
			name: "Broker with provider ID",
			token: &Token{
				Name:        "Broker Token",
				Role:        RoleBroker,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				BrokerID:    &validID,
				ProviderID:  &validID,
			},
			wantErr:    true,
			errMessage: "broker tokens should only have broker ID set",
		},
		{
			name: "Agent without agent ID",
			token: &Token{
				Name:        "Agent Token",
				Role:        RoleAgent,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				ProviderID:  &validID,
			},
			wantErr:    true,
			errMessage: "agent ID is required for agent role",
		},
		{
			name: "Agent without provider ID",
			token: &Token{
				Name:        "Agent Token",
				Role:        RoleAgent,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				AgentID:     &validID,
			},
			wantErr:    true,
			errMessage: "provider ID is required for agent role",
		},
		{
			name: "Agent with broker ID",
			token: &Token{
				Name:        "Agent Token",
				Role:        RoleAgent,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				AgentID:     &validID,
				ProviderID:  &validID,
				BrokerID:    &validID,
			},
			wantErr:    true,
			errMessage: "agent tokens should only have agent and provider ID's set",
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
