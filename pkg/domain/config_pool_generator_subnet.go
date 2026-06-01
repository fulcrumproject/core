package domain

import (
	"context"
	"fmt"
	"net"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ConfigPoolSubnetGenerator carves a parent CIDR. Without a prefix it allocates the
// next free host IP (scalar string). With a prefix it allocates the next free aligned
// /prefix block as a JSON object {cidr, fulcrumIp, cspIp, prefix}, pre-computing the
// two host addresses so config templates need no IP math.
type ConfigPoolSubnetGenerator struct {
	repo   ConfigPoolValueRepository
	poolID properties.UUID
	config properties.JSON
}

func NewConfigPoolSubnetGenerator(repo ConfigPoolValueRepository, poolID properties.UUID, config properties.JSON) *ConfigPoolSubnetGenerator {
	return &ConfigPoolSubnetGenerator{repo: repo, poolID: poolID, config: config}
}

func (g *ConfigPoolSubnetGenerator) Allocate(ctx context.Context, entityType ConfigPoolValueEntityType, entityID properties.UUID, propertyName string) (any, error) {
	sc, err := parseSubnetConfig(g.config)
	if err != nil {
		return nil, err
	}

	existing, err := g.repo.FindByPool(ctx, g.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pool values: %w", err)
	}
	used := usedSubnetKeys(existing)

	name, value, ok := sc.nextFree(used)
	if !ok {
		return nil, NewInvalidInputErrorf("subnet exhausted: no available values in pool")
	}

	cv := &ConfigPoolValue{Name: name, Value: value, ConfigPoolID: g.poolID}
	cv.Allocate(entityType, entityID, propertyName)
	if err := g.repo.Create(ctx, cv); err != nil {
		return nil, fmt.Errorf("failed to allocate value: %w", err)
	}
	return cv.RawValue(), nil
}

func (g *ConfigPoolSubnetGenerator) Release(ctx context.Context, values []*ConfigPoolValue) error {
	return releasePoolValues(ctx, g.repo, g.poolID, values)
}

type subnetConfig struct {
	base         net.IP
	ones, bits   int
	prefix       int
	hasPrefix    bool
	excludeFirst int
	excludeLast  int
	exclude      map[string]bool
}

// nextFree returns the (name, value) of the first free host or block not in used.
func (sc *subnetConfig) nextFree(used map[string]bool) (string, any, bool) {
	if sc.hasPrefix {
		blockSize := 1 << uint(sc.bits-sc.prefix)
		totalBlocks := 1 << uint(sc.prefix-sc.ones)
		for b := 0; b < totalBlocks; b++ {
			netAddr := incrementIP(sc.base, b*blockSize)
			cidr := fmt.Sprintf("%s/%d", netAddr.String(), sc.prefix)
			if used[cidr] {
				continue
			}
			value := map[string]any{
				"cidr":      cidr,
				"fulcrumIp": incrementIP(netAddr, 1).String(),
				"cspIp":     incrementIP(netAddr, 2).String(),
				"prefix":    sc.prefix,
			}
			return cidr, value, true
		}
		return "", nil, false
	}

	totalIPs := 1 << uint(sc.bits-sc.ones)
	for i := sc.excludeFirst; i < totalIPs-sc.excludeLast; i++ {
		ip := incrementIP(sc.base, i).String()
		if sc.exclude[ip] || used[ip] {
			continue
		}
		return ip, ip, true
	}
	return "", nil, false
}

// usedSubnetKeys extracts the identifying key of each existing value: the scalar
// string in host mode, or the "cidr" field in block mode.
func usedSubnetKeys(values []*ConfigPoolValue) map[string]bool {
	used := make(map[string]bool, len(values))
	for _, v := range values {
		switch val := v.Value.(type) {
		case string:
			used[val] = true
		case map[string]any:
			if cidr, ok := val["cidr"].(string); ok {
				used[cidr] = true
			}
		}
	}
	return used
}

func validateSubnetGeneratorConfig(cfg properties.JSON) error {
	_, err := parseSubnetConfig(cfg)
	return err
}

func parseSubnetConfig(cfg properties.JSON) (*subnetConfig, error) {
	cidr, ok := cfg["cidr"].(string)
	if !ok || cidr == "" {
		return nil, fmt.Errorf("subnet generator config requires string 'cidr'")
	}
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("subnet generator config 'cidr' is invalid: %w", err)
	}
	base := ip.Mask(ipNet.Mask).To4()
	if base == nil {
		return nil, fmt.Errorf("subnet generator config 'cidr' must be IPv4")
	}
	ones, bits := ipNet.Mask.Size()
	sc := &subnetConfig{base: base, ones: ones, bits: bits, exclude: map[string]bool{}}

	if p, ok := toInt(cfg["prefix"]); ok {
		if p < ones || p > bits {
			return nil, fmt.Errorf("subnet generator config 'prefix' (%d) must be between %d and %d", p, ones, bits)
		}
		sc.prefix = p
		sc.hasPrefix = true
	}
	if n, ok := toInt(cfg["excludeFirst"]); ok {
		sc.excludeFirst = n
	}
	if n, ok := toInt(cfg["excludeLast"]); ok {
		sc.excludeLast = n
	}
	if raw, present := cfg["exclude"]; present {
		list, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("subnet generator config 'exclude' must be an array")
		}
		for _, e := range list {
			s, ok := e.(string)
			if !ok || net.ParseIP(s) == nil {
				return nil, fmt.Errorf("subnet generator config 'exclude' entries must be IP strings")
			}
			sc.exclude[s] = true
		}
	}
	return sc, nil
}

var _ ConfigPoolGenerator = (*ConfigPoolSubnetGenerator)(nil)
