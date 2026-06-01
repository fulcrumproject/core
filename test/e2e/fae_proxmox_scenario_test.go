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

// testFaeProxmoxScenario exercises the range + subnet generators end to end across
// both pool scopes: ASN and PtP /30 are Fulcrum-global pools (admin, no participantId),
// while public IP, VMBR0 LAN /24 and GRE NAT /23 are CSP/participant-scoped. Onboarding
// GRE and L2 Proxmox clusters auto-allocates from both scopes in a single allocation,
// with values reused lowest-first after release.
func testFaeProxmoxScenario(t *testing.T, env *Env) {
	providerID := testhelpers.ProviderID
	uniq := testhelpers.Uniq()

	asnType := "fae-asn-" + uniq
	ptpType := "fae-ptp-" + uniq
	pubType := "fae-pubip-" + uniq
	lanType := "fae-lan-" + uniq
	natType := "fae-nat-" + uniq

	// Global pools (admin, no participantId): Fulcrum core network.
	asnPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.AdminClient, "/config-pools", api.CreateConfigPoolReq{
		Name:            "asn-" + uniq,
		Type:            asnType,
		PropertyType:    "integer",
		GeneratorType:   domain.PoolGeneratorRange,
		GeneratorConfig: &properties.JSON{"min": 65000, "max": 65535, "exclude": []any{65500}},
	})
	ptpPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.AdminClient, "/config-pools", api.CreateConfigPoolReq{
		Name:            "ptp-" + uniq,
		Type:            ptpType,
		PropertyType:    "json",
		GeneratorType:   domain.PoolGeneratorSubnet,
		GeneratorConfig: &properties.JSON{"cidr": "10.255.1.0/24", "prefix": 30},
	})

	// CSP pools (participant-scoped): the "forniti da CSP" parameters.
	pubPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
		Name:            "pubip-" + uniq,
		Type:            pubType,
		PropertyType:    "string",
		GeneratorType:   domain.PoolGeneratorSubnet,
		GeneratorConfig: &properties.JSON{"cidr": "212.78.11.0/24", "excludeFirst": 2, "excludeLast": 1},
		ParticipantID:   &providerID,
	})
	lanPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
		Name:            "lan-" + uniq,
		Type:            lanType,
		PropertyType:    "json",
		GeneratorType:   domain.PoolGeneratorSubnet,
		GeneratorConfig: &properties.JSON{"cidr": "10.255.242.0/23", "prefix": 24, "hosts": map[string]any{"gateway": 1}},
		ParticipantID:   &providerID,
	})
	natPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
		Name:            "nat-" + uniq,
		Type:            natType,
		PropertyType:    "json",
		GeneratorType:   domain.PoolGeneratorSubnet,
		GeneratorConfig: &properties.JSON{"cidr": "172.30.232.0/23", "prefix": 23, "hosts": map[string]any{}},
		ParticipantID:   &providerID,
	})

	poolProp := func(poolType string) schema.PropertyDefinition {
		return schema.PropertyDefinition{Generator: &schema.GeneratorConfig{Type: "pool", Config: map[string]any{"poolType": poolType}}}
	}
	greType := testhelpers.MustPost[api.CreateInfrastructureTypeReq, api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", api.CreateInfrastructureTypeReq{
		Name: "fae-gre-" + uniq,
		ConfigurationSchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{
			"transport": {Type: "string", Default: "gre", Immutable: true},
			"asn":       withType(poolProp(asnType), "integer"), // global
			"ptpSubnet": withType(poolProp(ptpType), "json"),    // global
			"publicIp":  withType(poolProp(pubType), "string"),  // CSP
			"lanSubnet": withType(poolProp(lanType), "json"),    // CSP
			"natSubnet": withType(poolProp(natType), "json"),    // CSP
		}},
	})
	l2Type := testhelpers.MustPost[api.CreateInfrastructureTypeReq, api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", api.CreateInfrastructureTypeReq{
		Name: "fae-l2-" + uniq,
		ConfigurationSchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{
			"transport": {Type: "string", Default: "l2", Immutable: true},
			"asn":       withType(poolProp(asnType), "integer"), // global
			"ptpSubnet": withType(poolProp(ptpType), "json"),    // global
			"lanSubnet": withType(poolProp(lanType), "json"),    // CSP
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
		testhelpers.MustDelete(t, env.AdminClient, "/config-pools", asnPool.ID)
		testhelpers.MustDelete(t, env.AdminClient, "/config-pools", ptpPool.ID)
		testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", pubPool.ID)
		testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", lanPool.ID)
		testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", natPool.ID)
	})

	// GRE resolves from both scopes in one allocation: asn/ptp from the global
	// pools, publicIp/lanSubnet/natSubnet from the CSP-scoped pools.
	require.NotNil(t, gre.Configuration)
	greCfg := *gre.Configuration
	require.Equal(t, "gre", greCfg["transport"])
	require.EqualValues(t, 65000, greCfg["asn"], "asn from the global range pool (range minimum)")
	ptp := greCfg["ptpSubnet"].(map[string]any)
	require.Equal(t, "10.255.1.0/30", ptp["cidr"], "ptp from the global subnet pool")
	require.Equal(t, "10.255.1.1", ptp["host1"])
	require.Equal(t, "10.255.1.2", ptp["host2"])
	require.Equal(t, "212.78.11.2", greCfg["publicIp"], "publicIp from the CSP pool (first host after excludeFirst)")
	lan := greCfg["lanSubnet"].(map[string]any)
	require.Equal(t, "10.255.242.0/24", lan["cidr"], "lanSubnet from the CSP pool")
	require.Equal(t, "10.255.242.1", lan["gateway"], "custom host label honoured")
	nat := greCfg["natSubnet"].(map[string]any)
	require.Equal(t, "172.30.232.0/23", nat["cidr"], "natSubnet from the CSP pool")
	require.NotContains(t, nat, "host1", "empty hosts emits no host fields")

	// L2 also spans both scopes: asn/ptp (global) + lanSubnet (CSP), reused lowest-first.
	require.NotNil(t, l2.Configuration)
	l2Cfg := *l2.Configuration
	require.Equal(t, "l2", l2Cfg["transport"])
	require.EqualValues(t, 65001, l2Cfg["asn"], "second cluster gets the next free ASN from the global pool")
	require.Equal(t, "10.255.1.4/30", l2Cfg["ptpSubnet"].(map[string]any)["cidr"], "second cluster gets the next free /30 from the global pool")
	require.Equal(t, "10.255.243.0/24", l2Cfg["lanSubnet"].(map[string]any)["cidr"], "second cluster gets the next free /24 from the CSP pool")
	require.NotContains(t, l2Cfg, "publicIp", "L2 type has no publicIp property, so none is allocated")
	require.NotContains(t, l2Cfg, "natSubnet", "L2 type has no natSubnet property")
}

func withType(p schema.PropertyDefinition, t string) schema.PropertyDefinition {
	p.Type = t
	return p
}
