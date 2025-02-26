package service

import (
	"context"

	"fulcrumproject.org/core/internal/domain"
)

// ServiceOperationService handles service operations that require job creation
type ServiceOperationService struct {
	serviceRepo domain.ServiceRepository
	jobRepo     domain.JobRepository
}

// NewServiceOperationService creates a new ServiceOperationService
func NewServiceOperationService(
	serviceRepo domain.ServiceRepository,
	jobRepo domain.JobRepository,
) *ServiceOperationService {
	return &ServiceOperationService{
		// Composition: ServiceOperationService delegates to serviceRepo
		serviceRepo: serviceRepo,
		jobRepo:     jobRepo,
	}
}

// CreateService handles service creation and creates a job for the agent
func (s *ServiceOperationService) CreateService(ctx context.Context, service *domain.Service) error {
	// Create the service
	if err := s.serviceRepo.Create(ctx, service); err != nil {
		return err
	}

	// Create a job for the agent
	job := &domain.Job{
		Type:      domain.JobServiceCreate,
		State:     domain.JobPending,
		AgentID:   service.AgentID,
		ServiceID: service.ID,
		Priority:  1,
		RequestData: domain.JSON{
			"serviceId": service.ID.String(),
			"resources": service.Resources,
		},
	}

	return s.jobRepo.Create(ctx, job)
}

// UpdateService handles service updates and creates a job for the agent
func (s *ServiceOperationService) UpdateService(ctx context.Context, service *domain.Service) error {
	// Update the service
	if err := s.serviceRepo.Save(ctx, service); err != nil {
		return err
	}

	// Create a job for the agent
	job := &domain.Job{
		Type:      domain.JobServiceUpdate,
		State:     domain.JobPending,
		AgentID:   service.AgentID,
		ServiceID: service.ID,
		Priority:  1,
		RequestData: domain.JSON{
			"serviceId": service.ID.String(),
			"resources": service.Resources,
		},
	}

	return s.jobRepo.Create(ctx, job)
}

// DeleteService handles service deletion and creates a job for the agent
func (s *ServiceOperationService) DeleteService(ctx context.Context, serviceID domain.UUID) error {
	// First get the service to know which agent should handle the deletion
	service, err := s.serviceRepo.FindByID(ctx, serviceID)
	if err != nil {
		return err
	}

	// Create a job for the agent before deleting the service
	job := &domain.Job{
		Type:      domain.JobServiceDelete,
		State:     domain.JobPending,
		AgentID:   service.AgentID,
		ServiceID: service.ID,
		Priority:  1,
		RequestData: domain.JSON{
			"serviceId": service.ID.String(),
		},
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return err
	}

	// Delete the service
	return s.serviceRepo.Delete(ctx, serviceID)
}

// Delegate methods to fulfill the ServiceRepository interface

// FindByID delegates to the underlying repository
func (s *ServiceOperationService) FindByID(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	return s.serviceRepo.FindByID(ctx, id)
}

// List delegates to the underlying repository
func (s *ServiceOperationService) List(ctx context.Context, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
	return s.serviceRepo.List(ctx, req)
}

// CountByGroup delegates to the underlying repository
func (s *ServiceOperationService) CountByGroup(ctx context.Context, groupID domain.UUID) (int64, error) {
	return s.serviceRepo.CountByGroup(ctx, groupID)
}

// Create is handled by CreateService
func (s *ServiceOperationService) Create(ctx context.Context, service *domain.Service) error {
	return s.CreateService(ctx, service)
}

// Save is handled by UpdateService
func (s *ServiceOperationService) Save(ctx context.Context, service *domain.Service) error {
	return s.UpdateService(ctx, service)
}

// Delete is handled by DeleteService
func (s *ServiceOperationService) Delete(ctx context.Context, id domain.UUID) error {
	return s.DeleteService(ctx, id)
}
