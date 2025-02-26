package domain

import (
	"fmt"
)

// JobState represents the current state of a job
type JobState string

const (
	JobPending    JobState = "Pending"
	JobProcessing JobState = "Processing"
	JobCompleted  JobState = "Completed"
	JobFailed     JobState = "Failed"
)

// ParseJobState parses a string into a JobState
func ParseJobState(s string) (JobState, error) {
	state := JobState(s)
	if err := state.Validate(); err != nil {
		return "", err
	}
	return state, nil
}

// Validate checks if the job state is valid
func (s JobState) Validate() error {
	switch s {
	case JobPending, JobProcessing, JobCompleted, JobFailed:
		return nil
	}
	return fmt.Errorf("invalid job state: %s", s)
}
