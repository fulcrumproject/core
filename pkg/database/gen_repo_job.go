// Job repository implementation using GORM Gen
// Provides type-safe database operations for Job entities
package database

import (
	"context"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type GenJobRepository struct {
	q *Query
}

func NewGenJobRepository(db *gorm.DB) *GenJobRepository {
	return &GenJobRepository{q: Use(db)}
}

func (r *GenJobRepository) Create(ctx context.Context, entity *domain.Job) error {
	return r.q.Job.WithContext(ctx).Create(entity)
}

func (r *GenJobRepository) Save(ctx context.Context, entity *domain.Job) error {
	result, err := r.q.Job.WithContext(ctx).Where(r.q.Job.ID.Eq(entity.ID)).Updates(entity)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenJobRepository) Delete(ctx context.Context, id properties.UUID) error {
	result, err := r.q.Job.WithContext(ctx).Where(r.q.Job.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return domain.NotFoundError{Err: gorm.ErrRecordNotFound}
	}
	return nil
}

func (r *GenJobRepository) Get(ctx context.Context, id properties.UUID) (*domain.Job, error) {
	entity, err := r.q.Job.WithContext(ctx).
		Preload(r.q.Job.Agent).
		Preload(r.q.Job.Service).
		Where(r.q.Job.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenJobRepository) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	count, err := r.q.Job.WithContext(ctx).Where(r.q.Job.ID.Eq(id)).Count()
	return count > 0, err
}

func (r *GenJobRepository) Count(ctx context.Context) (int64, error) {
	return r.q.Job.WithContext(ctx).Count()
}

func (r *GenJobRepository) List(ctx context.Context, scope *auth.IdentityScope, pageReq *domain.PageReq) (*domain.PageRes[domain.Job], error) {
	query := r.q.Job.WithContext(ctx).
		Preload(r.q.Job.Agent).
		Preload(r.q.Job.Service)
	query = applyGenJobAuthz(query, scope)

	result, err := PaginateQuery(ctx, query, pageReq,
		applyGenJobFilters,
		applyGenJobSort,
	)
	if err != nil {
		return nil, err
	}
	items := make([]domain.Job, len(result.Items))
	for i, item := range result.Items {
		items[i] = *item
	}
	return &domain.PageRes[domain.Job]{
		Items: items, TotalItems: result.TotalItems, TotalPages: result.TotalPages,
		CurrentPage: result.CurrentPage, HasNext: result.HasNext, HasPrev: result.HasPrev,
	}, nil
}

func (r *GenJobRepository) GetPendingJobsForAgent(ctx context.Context, agentID properties.UUID, limit int) ([]*domain.Job, error) {
	var jobs []*domain.Job
	
	// Use underlying DB for complex window function query
	processingGroupsSubquery := r.q.Job.WithContext(ctx).UnderlyingDB().
		Table("jobs").
		Select("DISTINCT services.group_id").
		Joins("JOIN services ON jobs.service_id = services.id").
		Where("jobs.agent_id = ? AND jobs.status = ?", agentID, domain.JobProcessing)

	subquery := r.q.Job.WithContext(ctx).UnderlyingDB().
		Table("jobs").
		Select("jobs.*, ROW_NUMBER() OVER (PARTITION BY services.group_id ORDER BY jobs.priority DESC, jobs.created_at ASC) as rn").
		Joins("JOIN services ON jobs.service_id = services.id").
		Where("jobs.agent_id = ? AND jobs.status = ?", agentID, domain.JobPending).
		Where("services.group_id NOT IN (?)", processingGroupsSubquery)

	err := r.q.Job.WithContext(ctx).UnderlyingDB().
		Preload("Service").
		Table("(?) as ranked_jobs", subquery).
		Where("ranked_jobs.rn = 1").
		Limit(limit).
		Find(&jobs).Error

	return jobs, err
}

func (r *GenJobRepository) GetLastJobForService(ctx context.Context, serviceID properties.UUID) (*domain.Job, error) {
	entity, err := r.q.Job.WithContext(ctx).
		Preload(r.q.Job.Agent).
		Preload(r.q.Job.Service).
		Where(r.q.Job.ServiceID.Eq(serviceID)).
		Order(r.q.Job.CreatedAt.Desc()).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	return entity, nil
}

func (r *GenJobRepository) GetTimeOutJobs(ctx context.Context, olderThan time.Duration) ([]*domain.Job, error) {
	cutoffTime := time.Now().Add(-olderThan)
	var timedOutJobs []*domain.Job
	err := r.q.Job.WithContext(ctx).UnderlyingDB().
		Where("status IN ? AND created_at < ?", []domain.JobStatus{domain.JobProcessing, domain.JobPending}, cutoffTime).
		Find(&timedOutJobs).Error
	return timedOutJobs, err
}

func (r *GenJobRepository) DeleteOldCompletedJobs(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-olderThan)
	result := r.q.Job.WithContext(ctx).UnderlyingDB().Exec(
		"DELETE FROM jobs WHERE (status = ? OR status = ?) AND completed_at < ?",
		domain.JobCompleted, domain.JobFailed, cutoffTime,
	)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

func (r *GenJobRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	job, err := r.q.Job.WithContext(ctx).
		Select(r.q.Job.AgentID).
		Where(r.q.Job.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NotFoundError{Err: err}
		}
		return nil, err
	}
	
	agent, err := r.q.Agent.WithContext(ctx).
		Select(r.q.Agent.ProviderID, r.q.Agent.ID).
		Where(r.q.Agent.ID.Eq(job.AgentID)).First()
	if err != nil {
		return nil, err
	}
	
	return &auth.DefaultObjectScope{
		ProviderID: &agent.ProviderID,
		AgentID:    &agent.ID,
	}, nil
}

func applyGenJobAuthz(query IJobDo, scope *auth.IdentityScope) IJobDo {
	q := Use(nil).Job
	qs := Use(nil).Service
	
	if scope.ParticipantID != nil {
		serviceIDs, _ := qs.WithContext(context.Background()).
			Select(qs.ID).
			Where(qs.ConsumerID.Eq(*scope.ParticipantID)).
			Or(qs.ProviderID.Eq(*scope.ParticipantID)).
			Find()
		if len(serviceIDs) > 0 {
			conditions := make([]gen.Condition, len(serviceIDs))
			for i, svc := range serviceIDs {
				conditions[i] = q.ServiceID.Eq(svc.ID)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}
	if scope.AgentID != nil {
		return query.Where(q.AgentID.Eq(*scope.AgentID))
	}
	return query
}

func applyGenJobFilters(query IJobDo, pageReq *domain.PageReq) IJobDo {
	q := Use(nil).Job
	
	if values, ok := pageReq.Filters["action"]; ok && len(values) > 0 {
		query = query.Where(q.Action.In(values...))
	}
	if values, ok := pageReq.Filters["status"]; ok && len(values) > 0 {
		statuses := make([]string, 0, len(values))
		for _, v := range values {
			if status, err := domain.ParseJobStatus(v); err == nil {
				statuses = append(statuses, string(status))
			}
		}
		if len(statuses) > 0 {
			query = query.Where(q.Status.In(statuses...))
		}
	}
	if values, ok := pageReq.Filters["agentId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.AgentID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}
	if values, ok := pageReq.Filters["serviceId"]; ok && len(values) > 0 {
		ids := parseUUIDs(values)
		if len(ids) > 0 {
			conditions := make([]gen.Condition, len(ids))
			for i, id := range ids {
				conditions[i] = q.ServiceID.Eq(id)
			}
			query = query.Where(conditions[0])
			if len(conditions) > 1 {
				query = query.Or(conditions[1:]...)
			}
		}
	}
	return query
}

func applyGenJobSort(query IJobDo, pageReq *domain.PageReq) IJobDo {
	if !pageReq.Sort {
		return query
	}
	q := Use(nil).Job
	switch pageReq.SortBy {
	case "priority":
		if pageReq.SortAsc {
			return query.Order(q.Priority)
		}
		return query.Order(q.Priority.Desc())
	case "createdAt":
		if pageReq.SortAsc {
			return query.Order(q.CreatedAt)
		}
		return query.Order(q.CreatedAt.Desc())
	case "claimedAt":
		if pageReq.SortAsc {
			return query.Order(q.ClaimedAt)
		}
		return query.Order(q.ClaimedAt.Desc())
	case "completedAt":
		if pageReq.SortAsc {
			return query.Order(q.CompletedAt)
		}
		return query.Order(q.CompletedAt.Desc())
	}
	return query
}

