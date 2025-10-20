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
	generatorConfig properties.JSON
}

// NewSubnetGenerator creates a new subnet-based generator
func NewSubnetGenerator(valueRepo ServicePoolValueRepository, poolID properties.UUID, config properties.JSON) *SubnetGenerator {
	return &SubnetGenerator{
		valueRepo:       valueRepo,
		poolID:          poolID,
		generatorConfig: config,
	}
}

// Allocate allocates the next available IP from the subnet
func (g *SubnetGenerator) Allocate(ctx context.Context, poolID properties.UUID, serviceID properties.UUID, propertyName string) (any, error) {
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

	// Get all existing values for this pool (to know which IPs are already allocated)
	existingValues, err := g.valueRepo.FindByPool(ctx, g.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing pool values: %w", err)
	}

	// Build a set of allocated IPs
	allocatedIPs := make(map[string]bool)
	for _, v := range existingValues {
		// Value can be stored as a plain string
		if ipStr, ok := v.Value.(string); ok {
			allocatedIPs[ipStr] = true
		}
	}

	// Calculate available IP range
	firstIP := ip.Mask(ipNet.Mask)
	ones, bits := ipNet.Mask.Size()
	totalIPs := 1 << uint(bits-ones)

	// Find next available IP
	var nextIP net.IP
	for i := excludeFirst; i < totalIPs-excludeLast; i++ {
		candidateIP := incrementIP(firstIP, i)
		if !ipNet.Contains(candidateIP) {
			continue
		}

		ipStr := candidateIP.String()
		if !allocatedIPs[ipStr] {
			nextIP = candidateIP
			break
		}
	}

	if nextIP == nil {
		return nil, NewInvalidInputErrorf("subnet exhausted: no available IPs in pool")
	}

	// Create new ServicePoolValue with the allocated IP
	ipStr := nextIP.String()
	now := time.Now()

	newValue := &ServicePoolValue{
		Name:          ipStr,
		Value:         ipStr, // Store IP address as a plain string
		ServicePoolID: g.poolID,
		ServiceID:     &serviceID,
		PropertyName:  &propertyName,
		AllocatedAt:   &now,
	}

	// Create the value in the repository
	err = g.valueRepo.Create(ctx, newValue)
	if err != nil {
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

		value.ServiceID = nil
		value.PropertyName = nil
		value.AllocatedAt = nil

		err = g.valueRepo.Update(ctx, value)
		if err != nil {
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
