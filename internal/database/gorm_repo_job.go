package database

import (
	"context"
	"time"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormJobRepository struct {
	*GormRepository[domain.Job]
}

var applyJobFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"action":    parserInFilterFieldApplier("action", domain.ParseServiceAction),
	"state":     parserInFilterFieldApplier("state", domain.ParseJobState),
	"agentId":   parserInFilterFieldApplier("agent_id", domain.ParseUUID),
	"serviceId": parserInFilterFieldApplier("service_id", domain.ParseUUID),
})

var applyJobSort = mapSortApplier(map[string]string{
	"priority":    "priority",
	"createdAt":   "created_at",
	"claimedAt":   "claimed_at",
	"completedAt": "completed_at",
})

// NewJobRepository creates a new instance of JobRepository
func NewJobRepository(db *gorm.DB) domain.JobRepository {
	repo := &gormJobRepository{
		GormRepository: NewGormRepository[domain.Job](
			db,
			applyJobFilter,
			applyJobSort,
			[]string{"Agent", "Service"}, // Find preload paths
			[]string{"Agent", "Service"}, // List preload paths - empty for performance reasons
		),
	}
	return repo
}

// GetPendingJobsForAgent retrieves pending jobs targeted for a specific agent
func (r *gormJobRepository) GetPendingJobsForAgent(ctx context.Context, agentID domain.UUID, limit int) ([]*domain.Job, error) {
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

// ReleaseStuckJobs resets jobs that have been processing for too long
func (r *gormJobRepository) ReleaseStuckJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-olderThan)
	result := r.db.WithContext(ctx).Exec(
		"UPDATE jobs SET state = ?, claimed_at = NULL WHERE state = ? AND claimed_at < ?",
		domain.JobPending, domain.JobProcessing, cutoffTime,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

// DeleteOldCompletedJobs removes completed or failed jobs older than the specified days
func (r *gormJobRepository) DeleteOldCompletedJobs(ctx context.Context, olderThan time.Duration) (int, error) {
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
