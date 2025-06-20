package api

import (
	"fmt"
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

func (t *JSONUTCTime) UnmarshalJSON(data []byte) error {
	// Remove quotes
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid time format")
	}
	timeStr := string(data[1 : len(data)-1])

	// Parse the time
	parsed, err := time.Parse(ISO8601UTC, timeStr)
	if err != nil {
		return err
	}

	*t = JSONUTCTime(parsed)
	return nil
}
