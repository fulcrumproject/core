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
	"state":     parserInFilterFieldApplier("jobs.state", domain.ParseJobState),
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
			jobAuthzFilterApplier,
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
		Where("agent_id = ? AND state = ?", agentID, domain.JobPending).
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
		Where("state IN ? AND created_at < ?", []domain.JobState{domain.JobProcessing, domain.JobPending}, cutoffTime).
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
		"DELETE FROM jobs WHERE (state = ? OR state = ?) AND completed_at < ?",
		domain.JobCompleted, domain.JobFailed, cutoffTime,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// jobAuthzFilterApplier applies authorization scoping to job queries
func jobAuthzFilterApplier(s *domain.AuthScope, q *gorm.DB) *gorm.DB {
	if s.ProviderID != nil {
		return q.Joins("INNER JOIN agents on agents.id = jobs.agent_id").Where("agents.provider_id", s.ProviderID)
	} else if s.BrokerID != nil {
		return q.Joins("INNER JOIN services ON services.id = jobs.service_id INNER JOIN service_groups on service_groups.id = services.group_id").Where("service_groups.broker_id", s.BrokerID)
	} else if s.AgentID != nil {
		return q.Where("agent_id = ?", s.AgentID)
	}
	return q
}
