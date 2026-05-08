//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func testServicePool(t *testing.T, env *Env) {
	name := "sp-" + testhelpers.Uniq()
	typeVal := "type_" + testhelpers.Uniq()
	created := testhelpers.MustPost[api.CreateServicePoolReq, api.ServicePoolRes](t, env.AdminClient, "/service-pools", api.CreateServicePoolReq{
		Name:             name,
		Type:             typeVal,
		PropertyType:     "string",
		GeneratorType:    domain.PoolGeneratorList,
		ServicePoolSetID: env.Seed.ServicePoolSet.ID,
	})
	require.Equal(t, name, created.Name)
	require.Equal(t, typeVal, created.Type)
	require.Equal(t, "string", created.PropertyType)
	require.Equal(t, domain.PoolGeneratorList, created.GeneratorType)
	require.Equal(t, env.Seed.ServicePoolSet.ID, created.ServicePoolSetID)
	require.NotEqual(t, properties.UUID{}, created.ID)
	require.False(t, time.Time(created.CreatedAt).IsZero())

	got := testhelpers.MustGet[api.ServicePoolRes](t, env.AdminClient, "/service-pools", created.ID)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Name, got.Name)
	require.Equal(t, created.Type, got.Type)
	require.Equal(t, created.PropertyType, got.PropertyType)
	require.Equal(t, created.GeneratorType, got.GeneratorType)
	require.Equal(t, created.ServicePoolSetID, got.ServicePoolSetID)

	newName := "sp-renamed-" + testhelpers.Uniq()
	updated := testhelpers.MustPatch[api.UpdateServicePoolReq, api.ServicePoolRes](t, env.AdminClient, "/service-pools", created.ID, api.UpdateServicePoolReq{Name: &newName})
	require.Equal(t, newName, updated.Name)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, created.ServicePoolSetID, updated.ServicePoolSetID, "PATCH name-only must not change FK")

	page := testhelpers.MustList[api.ServicePoolRes](t, env.AdminClient, "/service-pools")
	require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created service pool")

	testhelpers.MustDelete(t, env.AdminClient, "/service-pools", created.ID)
	testhelpers.AssertGone(t, env.AdminClient, "/service-pools", created.ID)
}
