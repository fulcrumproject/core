// Utility functions for validators
package schema

import (
	"encoding/json"
	"fmt"
)

// getIntConfig extracts and validates an integer configuration value
func getIntConfig(propPath, validatorName, configKey string, config map[string]any) (int, error) {
	value, ok := config[configKey]
	if !ok {
		return 0, fmt.Errorf("%s: %s validator requires '%s' config", propPath, validatorName, configKey)
	}

	var intValue int
	switch v := value.(type) {
	case int:
		intValue = v
	case float64:
		intValue = int(v)
	case int64:
		intValue = int(v)
	default:
		return 0, fmt.Errorf("%s: %s '%s' config must be a number", propPath, validatorName, configKey)
	}

	return intValue, nil
}

// getNonNegativeIntConfig extracts and validates a non-negative integer configuration value
func getNonNegativeIntConfig(propPath, validatorName, configKey string, config map[string]any) (int, error) {
	intValue, err := getIntConfig(propPath, validatorName, configKey, config)
	if err != nil {
		return 0, err
	}

	if intValue < 0 {
		return 0, fmt.Errorf("%s: %s value must be non-negative", propPath, validatorName)
	}

	return intValue, nil
}

// getFloatConfig extracts and validates a float configuration value
func getFloatConfig(propPath, validatorName, configKey string, config map[string]any) (float64, error) {
	value, ok := config[configKey]
	if !ok {
		return 0, fmt.Errorf("%s: %s validator requires '%s' config", propPath, validatorName, configKey)
	}

	var floatValue float64
	switch v := value.(type) {
	case int:
		floatValue = float64(v)
	case float64:
		floatValue = v
	case int64:
		floatValue = float64(v)
	default:
		return 0, fmt.Errorf("%s: %s '%s' config must be a number", propPath, validatorName, configKey)
	}

	return floatValue, nil
}

// convertToFloat64 converts any numeric type to float64
func convertToFloat64(propPath, validatorName string, value any) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case json.Number:
		return v.Float64()
	default:
		return 0, fmt.Errorf("%s: expected number for %s validator", propPath, validatorName)
	}
}
