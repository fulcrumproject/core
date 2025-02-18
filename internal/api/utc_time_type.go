package api

import (
	"time"
)

const (
	// ISO8601UTC is the standard time format used across the API
	ISO8601UTC = "2006-01-02T15:04:05Z07:00"
)

// JSONUTCTime is an UTC marshaled time
type JSONUTCTime time.Time

func (t JSONUTCTime) MarshalJSON() ([]byte, error) {
	formatted := time.Time(t).UTC().Format(ISO8601UTC)
	return []byte(`"` + formatted + `"`), nil
}
