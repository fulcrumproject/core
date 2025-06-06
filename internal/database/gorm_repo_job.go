package database

import (
	"context"
	"time"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormJobRepository struct {
	*GormRepository[domain.Job]
}

var applyJobFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"action":    parserInFilterFieldApplier("jobs.action", domain.ParseServiceAction),
	"status":    parserInFilterFieldApplier("jobs.status", domain.ParseJobStatus),
	"agentId":   parserInFilterFieldApplier("jobs.agent_id", domain.ParseUUID),
	"serviceId": parserInFilterFieldApplier("jobs.service_id", domain.ParseUUID),
})

var applyJobSort = mapSortApplier(map[string]string{
	"priority":    "jobs.priority",
	"createdAt":   "jobs.created_at",
	"claimedAt":   "jobs.claimed_at",
	"completedAt": "jobs.completed_at",
})

// NewJobRepository creates a new instance of JobRepository
func NewJobRepository(db *gorm.DB) *GormJobRepository {
	repo := &GormJobRepository{
		GormRepository: NewGormRepository[domain.Job](
			db,
			applyJobFilter,
			applyJobSort,
			providerConsumerAgentAuthzFilterApplier,
			[]string{"Agent", "Service"}, // Find preload paths
			[]string{"Agent", "Service"}, // List preload paths - empty for performance reasons
		),
	}
	return repo
}

// GetPendingJobsForAgent retrieves pending jobs targeted for a specific agent
func (r *GormJobRepository) GetPendingJobsForAgent(ctx context.Context, agentID domain.UUID, limit int) ([]*domain.Job, error) {
	var jobs []*domain.Job
	err := r.db.WithContext(ctx).
		Preload("Service").
		Where("agent_id = ? AND status = ?", agentID, domain.JobPending).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

// GetTimeOutJobs retrieves jobs that have been processing for too long and returns them
func (r *GormJobRepository) GetTimeOutJobs(ctx context.Context, olderThan time.Duration) ([]*domain.Job, error) {
	cutoffTime := time.Now().Add(-olderThan)

	var timedOutJobs []*domain.Job
	err := r.db.WithContext(ctx).
		Where("status IN ? AND created_at < ?", []domain.JobStatus{domain.JobProcessing, domain.JobPending}, cutoffTime).
		Find(&timedOutJobs).Error

	if err != nil {
		return nil, err
	}

	return timedOutJobs, nil
}

// DeleteOldCompletedJobs removes completed or failed jobs older than the specified days
func (r *GormJobRepository) DeleteOldCompletedJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-olderThan)
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM jobs WHERE (status = ? OR status = ?) AND completed_at < ?",
		domain.JobCompleted, domain.JobFailed, cutoffTime,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

func (r *GormJobRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	return r.getAuthScope(ctx, id, "provider_id", "consumer_id", "agent_id")
}
