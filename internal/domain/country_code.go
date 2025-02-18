package domain

import "errors"

var (
	ErrInvalidCountryCode = errors.New("invalid country code")
)

// CountryCode represents a validated ISO 3166-1 alpha-2 country code
type CountryCode string

// Validate ensures the CountryCode is a valid ISO 3166-1 alpha-2 code
func (c CountryCode) Validate() error {
	code := string(c)
	if len(code) != 2 {
		return ErrInvalidCountryCode
	}
	for _, ch := range code {
		if ch < 'A' || ch > 'Z' {
			return ErrInvalidCountryCode
		}
	}
	return nil
}

func ParseCountryCode(value string) (CountryCode, error) {
	code := CountryCode(value)
	return code, code.Validate()
}
