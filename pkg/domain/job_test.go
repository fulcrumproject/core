package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServiceAction_Validate(t *testing.T) {
	tests := []struct {
		name       string
		action     ServiceAction
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid ServiceActionCreate",
			action:  ServiceActionCreate,
			wantErr: false,
		},
		{
			name:    "Valid ServiceActionStart",
			action:  ServiceActionStart,
			wantErr: false,
		},
		{
			name:    "Valid ServiceActionStop",
			action:  ServiceActionStop,
			wantErr: false,
		},

		{
			name:    "Valid ServiceActionDelete",
			action:  ServiceActionDelete,
			wantErr: false,
		},
		{
			name:       "Invalid action",
			action:     "InvalidAction",
			wantErr:    true,
			errMessage: "invalid job type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseServiceAction(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       ServiceAction
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Parse ServiceActionCreate",
			input:   string(ServiceActionCreate),
			want:    ServiceActionCreate,
			wantErr: false,
		},
		{
			name:    "Parse ServiceActionStart",
			input:   string(ServiceActionStart),
			want:    ServiceActionStart,
			wantErr: false,
		},
		{
			name:    "Parse ServiceActionStop",
			input:   string(ServiceActionStop),
			want:    ServiceActionStop,
			wantErr: false,
		},

		{
			name:    "Parse ServiceActionDelete",
			input:   string(ServiceActionDelete),
			want:    ServiceActionDelete,
			wantErr: false,
		},
		{
			name:       "Parse invalid action",
			input:      "InvalidAction",
			wantErr:    true,
			errMessage: "invalid job type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServiceAction(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestJobStatus_Validate(t *testing.T) {
	tests := []struct {
		name       string
		status     JobStatus
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid JobPending",
			status:  JobPending,
			wantErr: false,
		},
		{
			name:    "Valid JobProcessing",
			status:  JobProcessing,
			wantErr: false,
		},
		{
			name:    "Valid JobCompleted",
			status:  JobCompleted,
			wantErr: false,
		},
		{
			name:    "Valid JobFailed",
			status:  JobFailed,
			wantErr: false,
		},
		{
			name:       "Invalid status",
			status:     "InvalidStatus",
			wantErr:    true,
			errMessage: "invalid job status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseJobStatus(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       JobStatus
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Parse JobPending",
			input:   string(JobPending),
			want:    JobPending,
			wantErr: false,
		},
		{
			name:    "Parse JobProcessing",
			input:   string(JobProcessing),
			want:    JobProcessing,
			wantErr: false,
		},
		{
			name:    "Parse JobCompleted",
			input:   string(JobCompleted),
			want:    JobCompleted,
			wantErr: false,
		},
		{
			name:    "Parse JobFailed",
			input:   string(JobFailed),
			want:    JobFailed,
			wantErr: false,
		},
		{
			name:       "Parse invalid status",
			input:      "InvalidStatus",
			wantErr:    true,
			errMessage: "invalid job status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJobStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestJob_TableName(t *testing.T) {
	job := Job{}
	assert.Equal(t, "jobs", job.TableName())
}

func TestJob_Validate(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name       string
		job        *Job
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid job",
			job: &Job{
				Action:    ServiceActionCreate,
				Status:    JobPending,
				Priority:  1,
				AgentID:   validID,
				ServiceID: validID,
			},
			wantErr: false,
		},
		{
			name: "Invalid action",
			job: &Job{
				Action:    "InvalidAction",
				Status:    JobPending,
				Priority:  1,
				AgentID:   validID,
				ServiceID: validID,
			},
			wantErr:    true,
			errMessage: "invalid action",
		},
		{
			name: "Invalid status",
			job: &Job{
				Action:    ServiceActionCreate,
				Status:    "InvalidStatus",
				Priority:  1,
				AgentID:   validID,
				ServiceID: validID,
			},
			wantErr:    true,
			errMessage: "invalid status",
		},
		{
			name: "Invalid priority",
			job: &Job{
				Action:    ServiceActionCreate,
				Status:    JobPending,
				Priority:  0,
				AgentID:   validID,
				ServiceID: validID,
			},
			wantErr:    true,
			errMessage: "priority must be greater than 0",
		},
		{
			name: "Empty agent ID",
			job: &Job{
				Action:    ServiceActionCreate,
				Status:    JobPending,
				Priority:  1,
				AgentID:   uuid.Nil,
				ServiceID: validID,
			},
			wantErr:    true,
			errMessage: "agent ID cannot be empty",
		},
		{
			name: "Empty service ID",
			job: &Job{
				Action:    ServiceActionCreate,
				Status:    JobPending,
				Priority:  1,
				AgentID:   validID,
				ServiceID: uuid.Nil,
			},
			wantErr:    true,
			errMessage: "service ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewJob(t *testing.T) {
	agentID := uuid.New()
	providerID := uuid.New()
	consumerID := uuid.New()
	serviceID := uuid.New()

	service := &Service{
		BaseEntity: BaseEntity{
			ID: serviceID,
		},
		ProviderID: providerID,
		AgentID:    agentID,
		ConsumerID: consumerID,
	}

	action := ServiceActionCreate
	priority := 5

	job := NewJob(service, action, nil, priority)

	assert.Equal(t, consumerID, job.ConsumerID)
	assert.Equal(t, providerID, job.ProviderID)
	assert.Equal(t, agentID, job.AgentID)
	assert.Equal(t, serviceID, job.ServiceID)
	assert.Equal(t, JobPending, job.Status)
	assert.Equal(t, action, job.Action)
	assert.Equal(t, priority, job.Priority)
}

func TestJob_Unsupported(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus JobStatus
		errorMessage  string
		wantErr       bool
		errMessage    string
	}{
		{
			name:          "Valid unsupported from processing",
			initialStatus: JobProcessing,
			errorMessage:  "Operation not supported by this agent",
			wantErr:       false,
		},
		{
			name:          "Invalid unsupported from pending",
			initialStatus: JobPending,
			errorMessage:  "Operation not supported",
			wantErr:       true,
			errMessage:    "cannot mark as unsupported a job not in processing status",
		},
		{
			name:          "Invalid unsupported from completed",
			initialStatus: JobCompleted,
			errorMessage:  "Operation not supported",
			wantErr:       true,
			errMessage:    "cannot mark as unsupported a job not in processing status",
		},
		{
			name:          "Invalid unsupported from failed",
			initialStatus: JobFailed,
			errorMessage:  "Operation not supported",
			wantErr:       true,
			errMessage:    "cannot mark as unsupported a job not in processing status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				Status: tt.initialStatus,
			}

			err := job.Unsupported(tt.errorMessage)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, JobUnsupported, job.Status)
				assert.Equal(t, tt.errorMessage, job.ErrorMessage)
				assert.NotNil(t, job.CompletedAt)
			}
		})
	}
}

func TestJobCommander_Unsupported(t *testing.T) {
	t.Run("Job not found", func(t *testing.T) {
		// Create mocks
		store := NewMockStore(t)
		jobRepo := NewMockJobRepository(t)

		// Setup mocks
		store.On("JobRepo").Return(jobRepo)
		jobRepo.On("Get", mock.Anything, mock.Anything).Return(nil, assert.AnError)

		// Create commander and execute test
		commander := NewJobCommander(store)
		params := UnsupportedJobParams{
			JobID:        uuid.New(),
			ErrorMessage: "Operation not supported",
		}

		err := commander.Unsupported(context.Background(), params)

		// Assert results
		assert.Error(t, err)
	})

	t.Run("Service not found", func(t *testing.T) {
		// Create mocks
		store := NewMockStore(t)
		jobRepo := NewMockJobRepository(t)

		// Create test data
		jobID := uuid.New()
		serviceID := uuid.New()

		job := &Job{
			BaseEntity: BaseEntity{ID: jobID},
			Status:     JobProcessing,
			ServiceID:  serviceID,
		}

		// Setup mocks
		store.On("JobRepo").Return(jobRepo)
		store.On("Atomic", mock.Anything, mock.AnythingOfType("func(domain.Store) error")).Return(assert.AnError)
		jobRepo.On("Get", mock.Anything, jobID).Return(job, nil)

		// Create commander and execute test
		commander := NewJobCommander(store)
		params := UnsupportedJobParams{
			JobID:        jobID,
			ErrorMessage: "Operation not supported",
		}

		err := commander.Unsupported(context.Background(), params)

		// Assert results
		assert.Error(t, err)
	})
}
