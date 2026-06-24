// Subnet-based pool generator implementation
package domain

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
)

// SubnetGenerator allocates IP addresses from a CIDR range
type SubnetGenerator struct {
	valueRepo       ServicePoolValueRepository
	poolID          properties.UUID
	participantID   *properties.UUID
	generatorConfig properties.JSON
}

// NewSubnetGenerator creates a new subnet-based generator
func NewSubnetGenerator(valueRepo ServicePoolValueRepository, poolID properties.UUID, participantID *properties.UUID, config properties.JSON) *SubnetGenerator {
	return &SubnetGenerator{
		valueRepo:       valueRepo,
		poolID:          poolID,
		participantID:   participantID,
		generatorConfig: config,
	}
}

// Allocate allocates the next available IP from the subnet
func (g *SubnetGenerator) Allocate(ctx context.Context, serviceID properties.UUID, propertyName string) (any, error) {
	// Parse generator config
	cidr, ok := g.generatorConfig["cidr"].(string)
	if !ok || cidr == "" {
		return nil, NewInvalidInputErrorf("invalid CIDR in generator config")
	}

	excludeFirst := 0
	if val, ok := g.generatorConfig["excludeFirst"].(float64); ok {
		excludeFirst = int(val)
	} else if val, ok := g.generatorConfig["excludeFirst"].(int); ok {
		excludeFirst = val
	}

	excludeLast := 0
	if val, ok := g.generatorConfig["excludeLast"].(float64); ok {
		excludeLast = int(val)
	} else if val, ok := g.generatorConfig["excludeLast"].(int); ok {
		excludeLast = val
	}

	// Parse CIDR
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, NewInvalidInputErrorf("invalid CIDR format: %v", err)
	}

	retention, err := parseRetention(g.generatorConfig)
	if err != nil {
		return nil, err
	}
	neverReallocate, err := parseNeverReallocate(g.generatorConfig)
	if err != nil {
		return nil, err
	}

	// Get all existing values for this pool
	existingValues, err := g.valueRepo.FindByPool(ctx, g.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing pool values: %w", err)
	}

	// Partition existing rows: allocated or still-cooling IPs are reserved (skipped);
	// freed IPs past their cooldown are reusable and re-allocated in place. With
	// neverReallocate a freed IP stays reserved permanently.
	now := time.Now()
	reserved := make(map[string]bool, len(existingValues))
	reusable := make(map[string]*ServicePoolValue, len(existingValues))
	for _, v := range existingValues {
		ipStr, ok := v.Value.(string)
		if !ok {
			continue
		}
		if v.IsAllocated() || !retentionAllows(retention, neverReallocate, v.ReleasedAt, now) {
			reserved[ipStr] = true
		} else {
			reusable[ipStr] = v
		}
	}

	// Calculate available IP range
	firstIP := ip.Mask(ipNet.Mask)
	ones, bits := ipNet.Mask.Size()
	totalIPs := 1 << uint(bits-ones)

	// Find the next free IP (not reserved)
	var nextIP net.IP
	for i := excludeFirst; i < totalIPs-excludeLast; i++ {
		candidateIP := incrementIP(firstIP, i)
		if !ipNet.Contains(candidateIP) {
			continue
		}
		if !reserved[candidateIP.String()] {
			nextIP = candidateIP
			break
		}
	}

	if nextIP == nil {
		return nil, NewInvalidInputErrorf("subnet exhausted: no available IPs in pool")
	}

	ipStr := nextIP.String()

	// Reuse a freed row in place when the chosen IP already has one; otherwise mint.
	if row, found := reusable[ipStr]; found {
		row.Allocate(serviceID, propertyName)
		if err := g.valueRepo.Update(ctx, row); err != nil {
			return nil, fmt.Errorf("failed to allocate value: %w", err)
		}
		return row.RawValue(), nil
	}

	newValue := &ServicePoolValue{
		Name:          ipStr,
		Value:         ipStr, // Store IP address as a plain string
		ServicePoolID: g.poolID,
		ParticipantID: g.participantID,
	}
	newValue.Allocate(serviceID, propertyName)
	if err := g.valueRepo.Create(ctx, newValue); err != nil {
		return nil, fmt.Errorf("failed to create allocated value: %w", err)
	}

	// Return the IP address string for copying to service property
	return ipStr, nil
}

// Release releases all allocations for the given service
func (g *SubnetGenerator) Release(ctx context.Context, serviceID properties.UUID) error {
	// Find all values allocated to this service
	allocatedValues, err := g.valueRepo.FindByService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to query allocated values: %w", err)
	}

	// Release each value from this pool
	for _, value := range allocatedValues {
		// Only release values from this pool
		if value.ServicePoolID != g.poolID {
			continue
		}

		value.Release()
		if err := g.valueRepo.Update(ctx, value); err != nil {
			return fmt.Errorf("failed to release value: %w", err)
		}
	}

	return nil
}

// incrementIP increments an IP address by n
func incrementIP(ip net.IP, n int) net.IP {
	result := make(net.IP, len(ip))
	copy(result, ip)

	// Convert to 4-byte representation if IPv4
	if len(result) == 16 && result.To4() != nil {
		result = result.To4()
	}

	// Increment from the least significant byte
	carry := n
	for i := len(result) - 1; i >= 0 && carry > 0; i-- {
		sum := int(result[i]) + carry
		result[i] = byte(sum & 0xFF)
		carry = sum >> 8
	}

	return result
}
