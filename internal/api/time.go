package api

import "time"

const (
	// ISO8601UTC is the standard time format used across the API
	ISO8601UTC = "2006-01-02T15:04:05Z07:00"
)

type JSONUTCTime time.Time

// MarshalJSON implements json.Marshaler interface
func (t JSONUTCTime) MarshalJSON() ([]byte, error) {
	formatted := time.Time(t).UTC().Format(ISO8601UTC)
	return []byte(`"` + formatted + `"`), nil
}
