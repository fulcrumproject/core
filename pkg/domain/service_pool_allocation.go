// Service pool allocation logic
package domain

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
)

// AllocateFromList allocates an available value from a list-type pool
// Returns the actual value to be copied to the service property
func AllocateFromList(
	ctx context.Context,
	repo ServicePoolValueRepository,
	poolID properties.UUID,
	serviceID properties.UUID,
	propertyName string,
) (any, error) {
	// Find available values
	availableValues, err := repo.FindAvailable(ctx, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available values: %w", err)
	}

	if len(availableValues) == 0 {
		return nil, NewInvalidInputErrorf("no available values in pool")
	}

	// Take the first available value
	value := availableValues[0]

	// Mark as allocated
	now := time.Now()
	value.ServiceID = &serviceID
	value.PropertyName = &propertyName
	value.AllocatedAt = &now

	// Update the value
	err = repo.Update(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate value: %w", err)
	}

	// Return the actual value for copying to service property
	return value.Value, nil
}

// AllocateFromSubnet allocates the next available IP from a subnet-type pool
// Returns the IP address value to be copied to the service property
func AllocateFromSubnet(
	ctx context.Context,
	repo ServicePoolValueRepository,
	poolID properties.UUID,
	serviceID properties.UUID,
	propertyName string,
	generatorConfig properties.JSON,
) (any, error) {
	// Parse generator config
	cidr, ok := generatorConfig["cidr"].(string)
	if !ok || cidr == "" {
		return nil, NewInvalidInputErrorf("invalid CIDR in generator config")
	}

	excludeFirst := 0
	if val, ok := generatorConfig["excludeFirst"].(float64); ok {
		excludeFirst = int(val)
	} else if val, ok := generatorConfig["excludeFirst"].(int); ok {
		excludeFirst = val
	}

	excludeLast := 0
	if val, ok := generatorConfig["excludeLast"].(float64); ok {
		excludeLast = int(val)
	} else if val, ok := generatorConfig["excludeLast"].(int); ok {
		excludeLast = val
	}

	// Parse CIDR
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, NewInvalidInputErrorf("invalid CIDR format: %v", err)
	}

	// Get all existing values for this pool (to know which IPs are already allocated)
	existingValues, err := repo.FindByPool(ctx, poolID)
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
		ServicePoolID: poolID,
		ServiceID:     &serviceID,
		PropertyName:  &propertyName,
		AllocatedAt:   &now,
	}

	// Create the value in the repository
	err = repo.Create(ctx, newValue)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocated value: %w", err)
	}

	// Return the IP address string for copying to service property
	return ipStr, nil
}

// ReleasePoolAllocations releases all pool allocations for a service
func ReleasePoolAllocations(
	ctx context.Context,
	repo ServicePoolValueRepository,
	serviceID properties.UUID,
) error {
	// Find all values allocated to this service
	allocatedValues, err := repo.FindByService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to query allocated values: %w", err)
	}

	// Release each value by clearing allocation fields
	for _, value := range allocatedValues {
		value.ServiceID = nil
		value.PropertyName = nil
		value.AllocatedAt = nil

		err = repo.Update(ctx, value)
		if err != nil {
			return fmt.Errorf("failed to release value %s: %w", value.ID, err)
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

// AllocatePoolProperty is a helper function that allocates from a pool based on its generator type
// Returns the actual value to be copied to the service property
func AllocatePoolProperty(
	ctx context.Context,
	poolRepo ServicePoolRepository,
	valueRepo ServicePoolValueRepository,
	poolID properties.UUID,
	serviceID properties.UUID,
	propertyName string,
) (any, error) {
	// Fetch the pool
	pool, err := poolRepo.Get(ctx, poolID)
	if err != nil {
		return nil, err
	}

	// Route to the appropriate generator based on pool type
	switch pool.GeneratorType {
	case PoolGeneratorList:
		return AllocateFromList(ctx, valueRepo, poolID, serviceID, propertyName)
	case PoolGeneratorSubnet:
		if pool.GeneratorConfig == nil {
			return nil, NewInvalidInputErrorf("subnet pool missing generator config")
		}
		return AllocateFromSubnet(ctx, valueRepo, poolID, serviceID, propertyName, *pool.GeneratorConfig)
	default:
		return nil, NewInvalidInputErrorf("unsupported pool generator type: %s", pool.GeneratorType)
	}
}
