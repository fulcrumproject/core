// Tests the optional CMP settings through Core's configuration builder.
// No dotenv files or real credentials are loaded by these cases.
package config

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/sso"
	"github.com/fulcrumproject/utils/confbuilder"
	"github.com/stretchr/testify/require"
)

func TestCmpSsoConfiguration(t *testing.T) {
	for _, name := range []string{"SIGNING_KEY", "KEY_ID", "ISSUER", "TOKEN_TTL"} {
		t.Setenv("FULCRUM_CMP_SSO_"+name, "")
	}
	t.Setenv("FULCRUM_PUBLIC_BASE_URL", "http://127.0.0.1:3000")
	cfg, err := confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.NoError(t, err)
	require.Equal(t, sso.DefaultTokenTTL, cfg.CmpSso.TokenTTL)
	signer, err := sso.New(cfg.CmpSso)
	require.NoError(t, err)
	require.Nil(t, signer)

	t.Setenv("FULCRUM_CMP_SSO_KEY_ID", "only-kid")
	cfg, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.NoError(t, err)
	_, err = sso.New(cfg.CmpSso)
	require.ErrorContains(t, err, "all required")

	t.Setenv("FULCRUM_CMP_SSO_SIGNING_KEY", "invalid PEM")
	t.Setenv("FULCRUM_CMP_SSO_ISSUER", "https://core.example")
	t.Setenv("FULCRUM_CMP_SSO_TOKEN_TTL", "45s")
	cfg, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.NoError(t, err)
	require.Equal(t, "invalid PEM", cfg.CmpSso.SigningKey)
	require.Equal(t, "only-kid", cfg.CmpSso.KeyID)
	require.Equal(t, "https://core.example", cfg.CmpSso.Issuer)
	require.Equal(t, 45*time.Second, cfg.CmpSso.TokenTTL)
	_, err = sso.New(cfg.CmpSso)
	require.ErrorContains(t, err, "parse CMP SSO signing key")
}
