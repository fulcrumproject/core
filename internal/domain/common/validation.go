package common

import (
	"errors"
	"regexp"
)

var (
	// ErrInvalidUUID indicates that the UUID is not valid
	ErrInvalidUUID = errors.New("invalid UUID")
	// ErrEmptyName indicates that the name field is empty
	ErrEmptyName = errors.New("name cannot be empty")
	// ErrInvalidCountryCode indicates that the country code is not valid
	ErrInvalidCountryCode = errors.New("invalid country code")
)

// CountryCodeRegex is a simple regex for ISO 3166-1 alpha-2 country codes
var CountryCodeRegex = regexp.MustCompile(`^[A-Z]{2}$`)

// ValidateUUID checks if a UUID is valid (not nil)
func ValidateUUID(id UUID) error {
	if id == (UUID{}) {
		return ErrInvalidUUID
	}
	return nil
}

// ValidateName checks if a name is not empty
func ValidateName(name string) error {
	if name == "" {
		return ErrEmptyName
	}
	return nil
}

// ValidateCountryCode checks if a country code is valid
func ValidateCountryCode(code string) error {
	if !CountryCodeRegex.MatchString(code) {
		return ErrInvalidCountryCode
	}
	return nil
}

// ValidateAttributes checks if attributes are valid
func ValidateAttributes(attrs Attributes) error {
	if attrs == nil {
		return nil
	}

	for key, values := range attrs {
		if key == "" {
			return errors.New("attribute key cannot be empty")
		}
		if values == nil {
			return errors.New("attribute values cannot be nil")
		}
	}
	return nil
}

// ValidateJSON checks if a JSON object is valid
func ValidateJSON(j JSON) error {
	if j == nil {
		return nil
	}

	for key := range j {
		if key == "" {
			return errors.New("JSON key cannot be empty")
		}
	}
	return nil
}
