package database

import (
	"context"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormJobRepository struct {
	*GormRepository[domain.Job]
}

var applyJobFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"action":    ParserInFilterFieldApplier("jobs.action", domain.ParseServiceAction),
	"status":    ParserInFilterFieldApplier("jobs.status", domain.ParseJobStatus),
	"agentId":   ParserInFilterFieldApplier("jobs.agent_id", properties.ParseUUID),
	"serviceId": ParserInFilterFieldApplier("jobs.service_id", properties.ParseUUID),
})

var applyJobSort = MapSortApplier(map[string]string{
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
// Returns only one pending job per service group with the highest priority
// Excludes service groups that have any jobs currently in processing status
func (r *GormJobRepository) GetPendingJobsForAgent(ctx context.Context, agentID properties.UUID, limit int) ([]*domain.Job, error) {
	var jobs []*domain.Job

	// Subquery to find service groups that have processing jobs
	processingGroupsSubquery := r.db.WithContext(ctx).
		Table("jobs").
		Select("DISTINCT services.group_id").
		Joins("JOIN services ON jobs.service_id = services.id").
		Where("jobs.agent_id = ? AND jobs.status = ?", agentID, domain.JobProcessing)

	// Use a subquery with window function to get the highest priority job per service group
	// Exclude service groups that have processing jobs
	subquery := r.db.WithContext(ctx).
		Table("jobs").
		Select("jobs.*, ROW_NUMBER() OVER (PARTITION BY services.group_id ORDER BY jobs.priority DESC, jobs.created_at ASC) as rn").
		Joins("JOIN services ON jobs.service_id = services.id").
		Where("jobs.agent_id = ? AND jobs.status = ?", agentID, domain.JobPending).
		Where("services.group_id NOT IN (?)", processingGroupsSubquery)

	err := r.db.WithContext(ctx).
		Preload("Service").
		Table("(?) as ranked_jobs", subquery).
		Where("ranked_jobs.rn = 1").
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

func (r *GormJobRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "agent_id", "consumer_id")
}
