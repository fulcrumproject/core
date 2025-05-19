package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJobCommander_Claim(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Claim success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}

				pendingJob := createJobWithState(jobID, JobPending)

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					assert.Equal(t, jobID, id)
					return pendingJob, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, JobProcessing, job.State)
					assert.NotNil(t, job.ClaimedAt)
					return nil
				}

				store.WithJobRepo(jobRepo)
			},
			wantErr: false,
		},
		{
			name: "Job not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return nil, NewNotFoundErrorf("job not found")
				}

				store.WithJobRepo(jobRepo)
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Job not in pending state",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}

				processingJob := createJobWithState(jobID, JobProcessing)

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return processingJob, nil
				}

				store.WithJobRepo(jobRepo)
			},
			wantErr:    true,
			errMessage: "cannot claim a job not in pending state",
		},
		{
			name: "Save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}

				pendingJob := createJobWithState(jobID, JobPending)

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return pendingJob, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, job *Job) error {
					return errors.New("save error")
				}

				store.WithJobRepo(jobRepo)
			},
			wantErr:    true,
			errMessage: "save error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewJobCommander(store, audit)
			err := commander.Claim(ctx, jobID)

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

func TestJobCommander_Complete(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()
	agentID := uuid.New()
	serviceID := uuid.New()
	providerID := uuid.New()
	brokerID := uuid.New()

	resources := JSON{"resource": "value"}
	externalID := "external-123"

	// Helper to create a processing job
	createProcessingJob := func() *Job {
		now := time.Now().Add(-1 * time.Hour)
		job := createJobWithState(jobID, JobProcessing)
		job.ServiceID = serviceID
		job.AgentID = agentID
		job.ProviderID = providerID
		job.ConsumerID = brokerID
		job.ClaimedAt = &now
		return job
	}

	// Helper to create a service with target state
	createService := func() *Service {
		currentState := ServiceStopped
		targetState := ServiceStarted
		return &Service{
			BaseEntity: BaseEntity{
				ID: serviceID,
			},
			Name:          "test-service",
			GroupID:       uuid.New(),
			ServiceTypeID: uuid.New(),
			CurrentState:  currentState,
			TargetState:   &targetState,
			AgentID:       agentID,
			ProviderID:    providerID,
			ConsumerID:    brokerID,
		}
	}

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		resources  *JSON
		externalID *string
		wantErr    bool
		errMessage string
	}{
		{
			name: "Complete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				// Set up necessary repositories
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				job := createProcessingJob()
				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					assert.Equal(t, jobID, id)
					return job, nil
				}

				service := createService()
				originalTargetState := ServiceStarted
				service.TargetState = &originalTargetState

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return service, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, updatedJob *Job) error {
					assert.Equal(t, JobCompleted, updatedJob.State)
					assert.NotNil(t, updatedJob.CompletedAt)
					return nil
				}

				serviceRepo.saveFunc = func(ctx context.Context, updatedService *Service) error {
					assert.Equal(t, updatedService.CurrentState, originalTargetState)
					assert.Nil(t, updatedService.TargetState)
					assert.Equal(t, &resources, updatedService.Resources)
					assert.Equal(t, &externalID, updatedService.ExternalID)
					return nil
				}

				// Set up audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					// Ensure the store used inside the function has the same repositories
					// This is critical as without this, the mock repos might be lost
					return fn(store)
				}
			},
			resources:  &resources,
			externalID: &externalID,
			wantErr:    false,
		},
		{
			name: "Job not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return nil, NewNotFoundErrorf("job not found")
				}

				store.WithJobRepo(jobRepo)
			},
			resources:  &resources,
			externalID: &externalID,
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Job not in processing state",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				pendingJob := createJobWithState(jobID, JobPending)
				pendingJob.ServiceID = serviceID

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return pendingJob, nil
				}

				service := createService()
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)
			},
			resources:  &resources,
			externalID: &externalID,
			wantErr:    true,
			errMessage: "cannot complete a job not in processing state",
		},
		{
			name: "Service not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return nil, NewNotFoundErrorf("service not found")
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)
			},
			resources:  &resources,
			externalID: &externalID,
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Service not in transition",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()

				service := createService()
				service.TargetState = nil

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)
			},
			resources:  &resources,
			externalID: &externalID,
			wantErr:    true,
			errMessage: "cannot complete a job on service that is not in transition",
		},
		{
			name: "Job save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()
				service := createService()

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, updatedJob *Job) error {
					return errors.New("job save error")
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			resources:  &resources,
			externalID: &externalID,
			wantErr:    true,
			errMessage: "job save error",
		},
		{
			name: "Service save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()
				service := createService()

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, updatedJob *Job) error {
					return nil
				}

				serviceRepo.saveFunc = func(ctx context.Context, updatedService *Service) error {
					return errors.New("service save error")
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			resources:  &resources,
			externalID: &externalID,
			wantErr:    true,
			errMessage: "service save error",
		},
		{
			name: "Audit error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()
				service := createService()

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, updatedJob *Job) error {
					return nil
				}

				serviceRepo.saveFunc = func(ctx context.Context, updatedService *Service) error {
					return nil
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)

				// Set up audit error
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return nil, errors.New("audit error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			resources:  &resources,
			externalID: &externalID,
			wantErr:    true,
			errMessage: "audit error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewJobCommander(store, audit)
			err := commander.Complete(ctx, jobID, tt.resources, tt.externalID)

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

func TestJobCommander_Fail(t *testing.T) {
	ctx := context.Background()
	jobID := uuid.New()
	agentID := uuid.New()
	serviceID := uuid.New()
	providerID := uuid.New()
	brokerID := uuid.New()

	errorMessage := "test error message"

	// Helper to create a processing job
	createProcessingJob := func() *Job {
		now := time.Now().Add(-1 * time.Hour)
		job := createJobWithState(jobID, JobProcessing)
		job.ServiceID = serviceID
		job.AgentID = agentID
		job.ProviderID = providerID
		job.ConsumerID = brokerID
		job.ClaimedAt = &now
		return job
	}

	// Helper to create a service
	createService := func() *Service {
		currentState := ServiceStopped
		targetState := ServiceStarted
		return &Service{
			BaseEntity: BaseEntity{
				ID: serviceID,
			},
			Name:          "test-service",
			GroupID:       uuid.New(),
			ServiceTypeID: uuid.New(),
			CurrentState:  currentState,
			TargetState:   &targetState,
			AgentID:       agentID,
			ProviderID:    providerID,
			ConsumerID:    brokerID,
		}
	}

	tests := []struct {
		name          string
		setupMocks    func(store *MockStore, audit *MockAuditEntryCommander)
		errorMessage  string
		wantErr       bool
		errMessageStr string
	}{
		{
			name: "Fail success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				job := createProcessingJob()
				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					assert.Equal(t, jobID, id)
					return job, nil
				}

				service := createService()

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return service, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, updatedJob *Job) error {
					assert.Equal(t, JobFailed, updatedJob.State)
					assert.Equal(t, errorMessage, updatedJob.ErrorMessage)
					return nil
				}

				serviceRepo.saveFunc = func(ctx context.Context, updatedService *Service) error {
					assert.Equal(t, &errorMessage, updatedService.ErrorMessage)
					assert.Equal(t, &job.Action, updatedService.FailedAction)
					return nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			errorMessage: errorMessage,
			wantErr:      false,
		},
		{
			name: "Job not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return nil, NewNotFoundErrorf("job not found")
				}

				store.WithJobRepo(jobRepo)
			},
			errorMessage:  errorMessage,
			wantErr:       true,
			errMessageStr: "not found",
		},
		{
			name: "Job not in processing state",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				pendingJob := createJobWithState(jobID, JobPending)
				pendingJob.ServiceID = serviceID

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return pendingJob, nil
				}

				service := createService()
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)
			},
			errorMessage:  errorMessage,
			wantErr:       true,
			errMessageStr: "cannot fail a job not in processing state",
		},
		{
			name: "Service not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return nil, NewNotFoundErrorf("service not found")
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)
			},
			errorMessage:  errorMessage,
			wantErr:       true,
			errMessageStr: "not found",
		},
		{
			name: "Job save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()
				service := createService()

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, updatedJob *Job) error {
					return errors.New("job save error")
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			errorMessage:  errorMessage,
			wantErr:       true,
			errMessageStr: "job save error",
		},
		{
			name: "Service save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()
				service := createService()

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, updatedJob *Job) error {
					return nil
				}

				serviceRepo.saveFunc = func(ctx context.Context, updatedService *Service) error {
					return errors.New("service save error")
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			errorMessage:  errorMessage,
			wantErr:       true,
			errMessageStr: "service save error",
		},
		{
			name: "Audit error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				jobRepo := &MockJobRepository{}
				serviceRepo := &MockServiceRepository{}

				job := createProcessingJob()
				service := createService()

				jobRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Job, error) {
					return job, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				jobRepo.saveFunc = func(ctx context.Context, updatedJob *Job) error {
					return nil
				}

				serviceRepo.saveFunc = func(ctx context.Context, updatedService *Service) error {
					return nil
				}

				store.WithJobRepo(jobRepo)
				store.WithServiceRepo(serviceRepo)

				// Set up audit error
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return nil, errors.New("audit error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			errorMessage:  errorMessage,
			wantErr:       true,
			errMessageStr: "audit error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewJobCommander(store, audit)
			err := commander.Fail(ctx, jobID, tt.errorMessage)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessageStr != "" {
					assert.Contains(t, err.Error(), tt.errMessageStr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
