package domain

import (
	"context"
	"errors"
)

type KeycloakUser struct {
	ID            string
	Username      string
	FirstName     string
	LastName      string
	Email         string
	Enabled       bool
	Roles         []string
	ParticipantID string
	AgentID       string
}

func (k *KeycloakUser) Validate() error {
	if k.ID == "" {
		return errors.New("keycloak user id is required")
	}

	if k.Username == "" {
		return errors.New("keycloak user username is required")
	}

	if k.Email == "" {
		return errors.New("keycloak user email is required")
	}

	if k.FirstName == "" {
		return errors.New("keycloak user first name is required")
	}

	if k.LastName == "" {
		return errors.New("keycloak user last name is required")
	}

	return nil
}

type KeycloakUserListParams struct {
	Search   string //maps to keycloak search contained in username, first or last name, or email.
	Page     int    // converted to "first" = (Page-1) * PageSize
	PageSize int    // maps to "max"
}

type KeycloakUserQuerier interface {
	List(ctx context.Context, params KeycloakUserListParams) ([]KeycloakUser, int, error)
}
