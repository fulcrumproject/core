// Tests generic JWT configuration through the existing Core config builder.
// Deprecated application-specific settings are not aliases for the issuer.
package config

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/jwtissuer"
	"github.com/fulcrumproject/utils/confbuilder"
	"github.com/stretchr/testify/require"
)

func TestJWTConfiguration(t *testing.T) {
	for _, name := range []string{"SIGNING_KEY", "KEY_ID", "ISSUER", "PARTICIPANT_TTL", "AGENT_TTL"} {
		t.Setenv("FULCRUM_JWT_"+name, "")
	}
	t.Setenv("FULCRUM_PUBLIC_BASE_URL", "http://127.0.0.1:3000")
	cfg, err := confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.NoError(t, err)
	require.Equal(t, jwtissuer.DefaultParticipantTTL, cfg.JWT.ParticipantTTL)
	require.Equal(t, jwtissuer.DefaultAgentTTL, cfg.JWT.AgentTTL)
	signer, err := jwtissuer.New(cfg.JWT)
	require.NoError(t, err)
	require.Nil(t, signer)
	for _, name := range []string{"CMP_SSO_SIGNING_KEY", "CMP_SSO_KEY_ID", "CMP_SSO_ISSUER", "CMP_SSO_TOKEN_TTL", "APPLICATION_TOKEN_TTL"} {
		t.Setenv("FULCRUM_"+name, "not-an-alias")
	}
	cfg, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.NoError(t, err)
	signer, err = jwtissuer.New(cfg.JWT)
	require.NoError(t, err)
	require.Nil(t, signer)
	t.Setenv("FULCRUM_JWT_KEY_ID", "only-kid")
	cfg, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.NoError(t, err)
	_, err = jwtissuer.New(cfg.JWT)
	require.ErrorContains(t, err, "all required")
	t.Setenv("FULCRUM_JWT_SIGNING_KEY", "invalid PEM")
	t.Setenv("FULCRUM_JWT_ISSUER", "https://core.example")
	t.Setenv("FULCRUM_JWT_PARTICIPANT_TTL", "45s")
	t.Setenv("FULCRUM_JWT_AGENT_TTL", "2m")
	cfg, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.NoError(t, err)
	require.Equal(t, 45*time.Second, cfg.JWT.ParticipantTTL)
	require.Equal(t, 2*time.Minute, cfg.JWT.AgentTTL)
	require.Equal(t, "invalid PEM", cfg.JWT.SigningKey)
	require.Equal(t, "only-kid", cfg.JWT.KeyID)
	require.Equal(t, "https://core.example", cfg.JWT.Issuer)
	_, err = jwtissuer.New(cfg.JWT)
	require.ErrorContains(t, err, "parse JWT signing key")
}
