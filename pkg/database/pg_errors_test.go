package database

import (
	"errors"
	"fmt"
	"testing"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslatePgError(t *testing.T) {
	tests := []struct {
		name     string
		in       error
		wantAs   any // pointer to target type for errors.As, or nil to expect passthrough
		wantMsg  string
		wantPass bool
	}{
		{
			name:    "unique violation with detail",
			in:      &pgconn.PgError{Code: "23505", Detail: "Key (name)=(foo) already exists.", ConstraintName: "agent_types_name_key"},
			wantAs:  &domain.ConflictError{},
			wantMsg: "name 'foo' already exists",
		},
		{
			name:    "unique violation without parseable detail stays generic",
			in:      &pgconn.PgError{Code: "23505", ConstraintName: "agent_types_name_key"},
			wantAs:  &domain.ConflictError{},
			wantMsg: "resource already exists",
		},
		{
			name:    "foreign key violation",
			in:      &pgconn.PgError{Code: "23503", Detail: "Key (config_pool_id)=(abc) is not present in table \"config_pools\"."},
			wantAs:  &domain.InvalidInputError{},
			wantMsg: "referenced config_pool_id 'abc' does not exist",
		},
		{
			name:    "vault duplicate reference maps to conflict",
			in:      &pgconn.PgError{Code: "23505", Detail: "Key (reference)=(foo) already exists.", ConstraintName: "vault_secrets_reference_key"},
			wantAs:  &domain.ConflictError{},
			wantMsg: "reference 'foo' already exists",
		},
		{
			name:    "not null violation",
			in:      &pgconn.PgError{Code: "23502", ColumnName: "name"},
			wantAs:  &domain.InvalidInputError{},
			wantMsg: "name is required",
		},
		{
			name:     "unmapped pg code passes through",
			in:       &pgconn.PgError{Code: "42P01"},
			wantPass: true,
		},
		{
			name:     "non-pg error passes through",
			in:       fmt.Errorf("boom"),
			wantPass: true,
		},
		{
			name:     "nil passes through",
			in:       nil,
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translatePgError(tt.in)
			if tt.wantPass {
				if got != tt.in {
					t.Fatalf("expected passthrough, got %v", got)
				}
				return
			}
			if !errors.As(got, tt.wantAs) {
				t.Fatalf("errors.As failed for %T, got %v", tt.wantAs, got)
			}
			if got.Error() != tt.wantMsg {
				t.Errorf("message = %q, want %q", got.Error(), tt.wantMsg)
			}
		})
	}
}
