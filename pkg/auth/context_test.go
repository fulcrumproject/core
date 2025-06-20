package auth

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithIdentity(t *testing.T) {
	testUUID := properties.NewUUID()
	identity := &Identity{
		ID:   testUUID,
		Name: "test-user",
		Role: RoleAdmin,
		Scope: IdentityScope{
			ParticipantID: nil,
			AgentID:       nil,
		},
	}

	ctx := context.Background()
	ctxWithIdentity := WithIdentity(ctx, identity)

	assert.NotNil(t, ctxWithIdentity, "Context with identity should not be nil")
	assert.NotEqual(t, ctx, ctxWithIdentity, "Context with identity should be different from original context")

	// Verify the identity can be retrieved
	retrievedIdentity := ctxWithIdentity.Value(identityContextKey)
	assert.Equal(t, identity, retrievedIdentity, "Retrieved identity should match the stored identity")
}

func TestMustGetIdentity(t *testing.T) {
	testUUID := properties.NewUUID()
	identity := &Identity{
		ID:   testUUID,
		Name: "test-user",
		Role: RoleParticipant,
		Scope: IdentityScope{
			ParticipantID: &testUUID,
			AgentID:       nil,
		},
	}

	ctx := WithIdentity(context.Background(), identity)
	retrievedIdentity := MustGetIdentity(ctx)

	require.NotNil(t, retrievedIdentity, "Retrieved identity should not be nil")
	assert.Equal(t, identity.ID, retrievedIdentity.ID, "Identity ID should match")
	assert.Equal(t, identity.Name, retrievedIdentity.Name, "Identity name should match")
	assert.Equal(t, identity.Role, retrievedIdentity.Role, "Identity role should match")
	assert.Equal(t, identity.Scope, retrievedIdentity.Scope, "Identity scope should match")
}

func TestMustGetIdentity_Panic_NoIdentity(t *testing.T) {
	ctx := context.Background()

	assert.Panics(t, func() {
		MustGetIdentity(ctx)
	}, "MustGetIdentity should panic when no identity is in context")
}

func TestMustGetIdentity_Panic_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), identityContextKey, "not-an-identity")

	assert.Panics(t, func() {
		MustGetIdentity(ctx)
	}, "MustGetIdentity should panic when context value is not an Identity")
}

func TestMustGetIdentity_Panic_NilIdentity(t *testing.T) {
	ctx := WithIdentity(context.Background(), nil)

	assert.Panics(t, func() {
		MustGetIdentity(ctx)
	}, "MustGetIdentity should panic when identity in context is nil")
}
