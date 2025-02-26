package domain

import (
	"fmt"
)

// JobType represents the type of operation a job performs
type JobType string

const (
	JobServiceCreate JobType = "ServiceCreate"
	JobServiceUpdate JobType = "ServiceUpdate"
	JobServiceDelete JobType = "ServiceDelete"
)

// ParseJobType parses a string into a JobType
func ParseJobType(s string) (JobType, error) {
	jobType := JobType(s)
	if err := jobType.Validate(); err != nil {
		return "", err
	}
	return jobType, nil
}

// Validate checks if the job type is valid
func (t JobType) Validate() error {
	switch t {
	case JobServiceCreate, JobServiceUpdate, JobServiceDelete:
		return nil
	}
	return fmt.Errorf("invalid job type: %s", t)
}
