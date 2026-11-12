// Tests application policy loading through the supported JSON file and env builder.
// No dotenv files or external credentials are accessed by these tests.
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/sso"
	"github.com/fulcrumproject/utils/confbuilder"
	"github.com/stretchr/testify/require"
)

func TestApplicationConfiguration(t *testing.T) {
	t.Setenv("FULCRUM_PUBLIC_BASE_URL", "http://127.0.0.1:3000")
	t.Setenv("FULCRUM_APPLICATION_TOKEN_TTL", "")
	cfg, err := confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.NoError(t, err)
	require.Equal(t, sso.DefaultApplicationTTL, cfg.ApplicationTokens.TokenTTL)
	policy, err := sso.NewApplicationTokens(nil, cfg.ApplicationTokens)
	require.NoError(t, err)
	require.False(t, policy.Enabled())
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"applicationTokens":{"tokenTtl":60000000000,"grants":{"33333333-3333-4333-8333-333333333333":{"audience":"fae-demo","roles":["pve"]}}}}`), 0600))
	cfg, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).File(&path).Build()
	require.NoError(t, err)
	require.Equal(t, time.Minute, cfg.ApplicationTokens.TokenTTL)
	require.Equal(t, sso.ApplicationGrant{Audience: "fae-demo", Roles: []string{"pve"}}, cfg.ApplicationTokens.Grants["33333333-3333-4333-8333-333333333333"])
	_, err = sso.NewApplicationTokens(nil, cfg.ApplicationTokens)
	require.ErrorContains(t, err, "requires CMP SSO signing settings")
	t.Setenv("FULCRUM_APPLICATION_TOKEN_TTL", "2m")
	cfg, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).File(&path).Build()
	require.NoError(t, err)
	require.Equal(t, 2*time.Minute, cfg.ApplicationTokens.TokenTTL)
	require.Equal(t, 90*time.Second, cfg.CmpSso.TokenTTL)
	t.Setenv("FULCRUM_APPLICATION_TOKEN_TTL", "invalid")
	_, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).Build()
	require.Error(t, err)
	t.Setenv("FULCRUM_APPLICATION_TOKEN_TTL", "")
	require.NoError(t, os.WriteFile(path, []byte(`{"applicationTokens":{"grants":[]}}`), 0600))
	_, err = confbuilder.New(Default).EnvPrefix(EnvPrefix).File(&path).Build()
	require.Error(t, err)
}
