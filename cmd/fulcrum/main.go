package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fulcrumproject/core/pkg/app"
)

func main() {
	application := app.NewApp()
	if application == nil {
		slog.Error("Failed to create app")
		os.Exit(1)
	}
	var jobMaintenanceWorker *app.JobMaintenanceWorker
	var agentsWorker *app.UnhealthyAgentsWorker

	if application.Config.JobMaintenance {
		jobMaintenanceWorker = app.NewJobMaintenanceWorker(application)
		if err := jobMaintenanceWorker.Run(); err != nil {
			slog.Error("Failed to run job maintenance worker", "error", err)
			os.Exit(1)
		}
	}

	if application.Config.AgentMaintenance {
		agentsWorker = app.NewUnhealthyAgentsWorker(application)
		if err := agentsWorker.Run(); err != nil {
			slog.Error("Failed to run agents worker", "error", err)
			os.Exit(1)
		}
	}

	var apiServer *app.ApiServer
	if application.Config.ApiServer {
		apiServer = app.NewApiServer(application)
		if apiServer == nil {
			slog.Error("Failed to create http app")
			os.Exit(1)
		}
		if err := apiServer.Start(); err != nil {
			slog.Error("Failed to start http app", "error", err)
			os.Exit(1)
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	slog.Info("Shutting down server...")

	if apiServer != nil {
		apiServer.Close()
	}

	if jobMaintenanceWorker != nil {
		jobMaintenanceWorker.Close()
	}

	if agentsWorker != nil {
		agentsWorker.Close()
	}
}
