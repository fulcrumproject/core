package domain

import (
	"context"
	"fmt"
	"net"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ConfigPoolSubnetGenerator carves a parent CIDR. Without a prefix it allocates the
// next free host IP (scalar string). With a prefix it allocates the next free aligned
// /prefix block as a JSON object {cidr, prefix} plus the named host addresses from the
// "hosts" config (defaulting to {host1, host2}), pre-computing them so config templates
// need no IP math.
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
	hosts        map[string]int
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
			value := map[string]any{"cidr": cidr, "prefix": sc.prefix}
			for name, off := range sc.hosts {
				value[name] = incrementIP(netAddr, off).String()
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
	if raw, present := cfg["hosts"]; present {
		if !sc.hasPrefix {
			return nil, fmt.Errorf("subnet generator config 'hosts' requires 'prefix' (block mode)")
		}
		obj, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("subnet generator config 'hosts' must be an object of name->offset")
		}
		blockSize := 1 << uint(sc.bits-sc.prefix)
		sc.hosts = make(map[string]int, len(obj))
		for name, v := range obj {
			if name == "" || name == "cidr" || name == "prefix" {
				return nil, fmt.Errorf("subnet generator config 'hosts' key %q is reserved or empty", name)
			}
			off, ok := toInt(v)
			if !ok || off < 0 || off >= blockSize {
				return nil, fmt.Errorf("subnet generator config 'hosts' offset for %q must be 0..%d", name, blockSize-1)
			}
			sc.hosts[name] = off
		}
	} else if sc.hasPrefix {
		// Default host labels, chosen to fit the carved block:
		//   /30 and larger -> {host1:1, host2:2} (skip network/broadcast)
		//   /31            -> {host1:0, host2:1} (RFC 3021, both ends usable)
		//   /32            -> none (single address; the cidr already names it)
		blockSize := 1 << uint(sc.bits-sc.prefix)
		switch {
		case blockSize >= 4:
			sc.hosts = map[string]int{"host1": 1, "host2": 2}
		case blockSize == 2:
			sc.hosts = map[string]int{"host1": 0, "host2": 1}
		default:
			sc.hosts = map[string]int{}
		}
	}

	// excludeFirst/excludeLast/exclude trim the usable host range and apply to
	// host mode only; block mode carves aligned blocks and has no host range.
	_, hasExcludeFirst := cfg["excludeFirst"]
	_, hasExcludeLast := cfg["excludeLast"]
	_, hasExclude := cfg["exclude"]
	if sc.hasPrefix && (hasExcludeFirst || hasExcludeLast || hasExclude) {
		return nil, fmt.Errorf("subnet generator config 'excludeFirst'/'excludeLast'/'exclude' apply to host mode only (no 'prefix')")
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

	if !sc.hasPrefix {
		totalIPs := 1 << uint(sc.bits-sc.ones)
		if sc.excludeFirst < 0 || sc.excludeLast < 0 {
			return nil, fmt.Errorf("subnet generator config 'excludeFirst' (%d) and 'excludeLast' (%d) must be >= 0", sc.excludeFirst, sc.excludeLast)
		}
		if sc.excludeFirst+sc.excludeLast >= totalIPs {
			return nil, fmt.Errorf("subnet generator config 'excludeFirst' (%d) + 'excludeLast' (%d) leave no addresses in the /%d subnet", sc.excludeFirst, sc.excludeLast, sc.ones)
		}
	}
	return sc, nil
}

var _ ConfigPoolGenerator = (*ConfigPoolSubnetGenerator)(nil)
