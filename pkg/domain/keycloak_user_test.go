package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeycloakUser_Validate(t *testing.T) {
	tests := []struct {
		name         string
		keycloakUser *KeycloakUser
		wantErr      bool
		errMessage   string
	}{
		{
			name: "Valid user",
			keycloakUser: &KeycloakUser{
				ID:        "some-kc-id",
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: false,
		},
		{
			name: "Missing id",
			keycloakUser: &KeycloakUser{
				ID:        "",
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr:    true,
			errMessage: "keycloak user id is required",
		},
		{
			name: "Missing username",
			keycloakUser: &KeycloakUser{
				ID:        "some-kc-id",
				Username:  "",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr:    true,
			errMessage: "keycloak user username is required",
		},
		{
			name: "Missing email",
			keycloakUser: &KeycloakUser{
				ID:        "some-kc-id",
				Username:  "some-username",
				Email:     "",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr:    true,
			errMessage: "keycloak user email is required",
		},
		{
			name: "Missing first name",
			keycloakUser: &KeycloakUser{
				ID:        "some-kc-id",
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "",
				LastName:  "Doe",
			},
			wantErr:    true,
			errMessage: "keycloak user first name is required",
		},
		{
			name: "Missing last name",
			keycloakUser: &KeycloakUser{
				ID:        "some-kc-id",
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "",
			},
			wantErr:    true,
			errMessage: "keycloak user last name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.keycloakUser.Validate()
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
