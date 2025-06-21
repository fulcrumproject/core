package helpers

import "github.com/fulcrumproject/core/pkg/properties"

// StringPtr returns a pointer to the given string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to the given bool
func BoolPtr(b bool) *bool {
	return &b
}

// JSONPtr returns a pointer to the given JSON
func JSONPtr(j properties.JSON) *properties.JSON {
	return &j
}
