//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

// testFaeProxmoxScenario exercises the range + subnet generators end to end:
// onboarding GRE and L2 Proxmox clusters auto-allocates an ASN, a PtP /30 (JSON)
// and — only for GRE — a public IP, with values reused lowest-first after release.
func testFaeProxmoxScenario(t *testing.T, env *Env) {
	providerID := testhelpers.ProviderID
	uniq := testhelpers.Uniq()

	asnType := "fae-asn-" + uniq
	ptpType := "fae-ptp-" + uniq
	pubType := "fae-pubip-" + uniq

	asnPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
		Name:            "asn-" + uniq,
		Type:            asnType,
		PropertyType:    "integer",
		GeneratorType:   domain.PoolGeneratorRange,
		GeneratorConfig: &properties.JSON{"min": 65000, "max": 65535, "exclude": []any{65500}},
		ParticipantID:   &providerID,
	})
	ptpPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
		Name:            "ptp-" + uniq,
		Type:            ptpType,
		PropertyType:    "json",
		GeneratorType:   domain.PoolGeneratorSubnet,
		GeneratorConfig: &properties.JSON{"cidr": "10.255.1.0/24", "prefix": 30},
		ParticipantID:   &providerID,
	})
	pubPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
		Name:            "pubip-" + uniq,
		Type:            pubType,
		PropertyType:    "string",
		GeneratorType:   domain.PoolGeneratorSubnet,
		GeneratorConfig: &properties.JSON{"cidr": "212.78.11.0/24", "excludeFirst": 2, "excludeLast": 1},
		ParticipantID:   &providerID,
	})

	poolProp := func(poolType string) schema.PropertyDefinition {
		return schema.PropertyDefinition{Generator: &schema.GeneratorConfig{Type: "pool", Config: map[string]any{"poolType": poolType}}}
	}
	greType := testhelpers.MustPost[api.CreateInfrastructureTypeReq, api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", api.CreateInfrastructureTypeReq{
		Name: "fae-gre-" + uniq,
		ConfigurationSchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{
			"transport": {Type: "string", Default: "gre", Immutable: true},
			"asn":       withType(poolProp(asnType), "integer"),
			"ptpSubnet": withType(poolProp(ptpType), "json"),
			"publicIp":  withType(poolProp(pubType), "string"),
		}},
	})
	l2Type := testhelpers.MustPost[api.CreateInfrastructureTypeReq, api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", api.CreateInfrastructureTypeReq{
		Name: "fae-l2-" + uniq,
		ConfigurationSchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{
			"transport": {Type: "string", Default: "l2", Immutable: true},
			"asn":       withType(poolProp(asnType), "integer"),
			"ptpSubnet": withType(poolProp(ptpType), "json"),
		}},
	})

	emptyCfg := properties.JSON{}
	gre := testhelpers.MustPost[api.CreateInfrastructureReq, api.InfrastructureRes](t, env.AdminClient, "/infrastructures", api.CreateInfrastructureReq{
		Name:                 "fae-gre-cluster-" + uniq,
		ProviderID:           providerID,
		InfrastructureTypeID: greType.ID,
		Configuration:        &emptyCfg,
	})
	l2 := testhelpers.MustPost[api.CreateInfrastructureReq, api.InfrastructureRes](t, env.AdminClient, "/infrastructures", api.CreateInfrastructureReq{
		Name:                 "fae-l2-cluster-" + uniq,
		ProviderID:           providerID,
		InfrastructureTypeID: l2Type.ID,
		Configuration:        &emptyCfg,
	})
	t.Cleanup(func() {
		testhelpers.MustDelete(t, env.AdminClient, "/infrastructures", gre.ID)
		testhelpers.MustDelete(t, env.AdminClient, "/infrastructures", l2.ID)
		testhelpers.MustDelete(t, env.AdminClient, "/infrastructure-types", greType.ID)
		testhelpers.MustDelete(t, env.AdminClient, "/infrastructure-types", l2Type.ID)
		testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", asnPool.ID)
		testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", ptpPool.ID)
		testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", pubPool.ID)
	})

	require.NotNil(t, gre.Configuration)
	greCfg := *gre.Configuration
	require.Equal(t, "gre", greCfg["transport"])
	require.EqualValues(t, 65000, greCfg["asn"], "first ASN is the range minimum")
	require.Equal(t, "212.78.11.2", greCfg["publicIp"], "first public host after excludeFirst")
	ptp := greCfg["ptpSubnet"].(map[string]any)
	require.Equal(t, "10.255.1.0/30", ptp["cidr"])
	require.Equal(t, "10.255.1.1", ptp["fulcrumIp"])
	require.Equal(t, "10.255.1.2", ptp["cspIp"])

	require.NotNil(t, l2.Configuration)
	l2Cfg := *l2.Configuration
	require.Equal(t, "l2", l2Cfg["transport"])
	require.EqualValues(t, 65001, l2Cfg["asn"], "second cluster gets the next free ASN")
	require.Equal(t, "10.255.1.4/30", l2Cfg["ptpSubnet"].(map[string]any)["cidr"], "second cluster gets the next free /30")
	require.NotContains(t, l2Cfg, "publicIp", "L2 type has no publicIp property, so none is allocated")
}

func withType(p schema.PropertyDefinition, t string) schema.PropertyDefinition {
	p.Type = t
	return p
}
