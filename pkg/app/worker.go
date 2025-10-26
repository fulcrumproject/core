package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/fulcrumproject/core/pkg/config"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-co-op/gocron/v2"
)

type UnhealthyAgentsWorker struct {
	app *App
}

func NewUnhealthyAgentsWorker(app *App) *UnhealthyAgentsWorker {
	return &UnhealthyAgentsWorker{
		app: app,
	}
}

func (w *UnhealthyAgentsWorker) Run() error {
	task := disconnectUnhealthyAgentsTask(&w.app.Config.AgentConfig, w.app.Store, w.app.WaitGroup)
	err := scheduleWork(task, w.app.Scheduler, w.app.Config.AgentConfig.HealthTimeout, "agent_maintenance")
	if err != nil {
		slog.Error("Failed to schedule work", "error", err)
		return err
	}
	w.app.StartScheduler()
	return nil
}

func (w *UnhealthyAgentsWorker) Close() {
	w.app.WaitGroup.Wait()
}

type JobMaintenanceWorker struct {
	app *App
}

func NewJobMaintenanceWorker(app *App) *JobMaintenanceWorker {
	return &JobMaintenanceWorker{
		app: app,
	}
}

func (w *JobMaintenanceWorker) Run() error {
	task := jobMaintenanceTask(&w.app.Config.JobConfig, w.app.Store, w.app.ServiceCmd, w.app.WaitGroup)
	err := scheduleWork(task, w.app.Scheduler, w.app.Config.JobConfig.Maintenance, "job_maintenance")
	if err != nil {
		slog.Error("Failed to schedule work", "error", err)
		return err
	}
	w.app.StartScheduler()
	return nil
}

func (w *JobMaintenanceWorker) Close() {
	w.app.WaitGroup.Wait()
}

func scheduleWork(task gocron.Task, scheduler *gocron.Scheduler, duration time.Duration, job_name string) error {

	j, err := (*scheduler).NewJob(
		gocron.DurationJob(duration),
		task,
		gocron.WithName(job_name),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		slog.Error("Failed to create job", "error", err)
		return err
	}

	slog.Info("Job ID", "id", j.ID())

	return nil
}

func disconnectUnhealthyAgentsTask(cfg *config.AgentConfig, store domain.Store, wg *sync.WaitGroup) gocron.Task {
	task := gocron.NewTask(
		func(cfg *config.AgentConfig, store domain.Store, wg *sync.WaitGroup) {
			wg.Add(1)
			defer wg.Done()
			ctx := context.Background()

			slog.Info("Checking agents health")
			disconnectedCount, err := store.AgentRepo().MarkInactiveAgentsAsDisconnected(ctx, cfg.HealthTimeout)
			if err != nil {
				slog.Error("Error marking inactive agents as disconnected", "error", err)
			} else if disconnectedCount > 0 {
				slog.Info("Marked inactive agents as disconnected", "count", disconnectedCount)
			}
		},
		cfg,
		store,
		wg,
	)

	return task
}

func jobMaintenanceTask(cfg *config.JobConfig, store domain.Store, serviceCmd domain.ServiceCommander, wg *sync.WaitGroup) gocron.Task {
	task := gocron.NewTask(
		func(cfg *config.JobConfig, store domain.Store, serviceCmd domain.ServiceCommander, wg *sync.WaitGroup) {
			wg.Add(1)
			defer wg.Done()
			ctx := context.Background()

			// Fail timeout jobs an services
			slog.Info("Checking timeout jobs")
			failedCount, err := serviceCmd.FailTimeoutServicesAndJobs(ctx, cfg.Timeout)
			if err != nil {
				slog.Error("Failed to timeout jobs and services", "error", err)
			} else {
				slog.Info("Timeout jobs processed", "failed_count", failedCount)
			}

			// Delete completed/failed old jobs
			slog.Info("Deleting old jobs")
			deletedCount, err := store.JobRepo().DeleteOldCompletedJobs(ctx, cfg.Retention)
			if err != nil {
				slog.Error("Failed to delete old jobs", "error", err)
			} else {
				slog.Info("Old jobs deleted", "count", deletedCount)
			}
		},
		cfg,
		store,
		serviceCmd,
		wg,
	)

	return task
}
