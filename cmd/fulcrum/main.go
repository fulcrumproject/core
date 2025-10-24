package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/config"
	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/gormlock"
	"github.com/fulcrumproject/core/pkg/health"
	"github.com/fulcrumproject/core/pkg/keycloak"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/utils/confbuilder"
	"github.com/fulcrumproject/utils/logging"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := confbuilder.New(config.Default).
		EnvPrefix(config.EnvPrefix).
		EnvFiles(".env").
		File(configPath).
		Build()
	if err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	// Setup structured logger
	logger := logging.NewLogger(&cfg.LogConfig)
	slog.SetDefault(logger)

	slog.Debug("API_SERVER", "value", cfg.ApiServer)
	slog.Debug("JOB_MAINTENANCE", "value", cfg.JobMaintenance)
	slog.Debug("AGENT_MAINTENANCE", "value", cfg.AgentMaintenance)

	// Initialize database
	db, err := database.NewConnection(&cfg.DBConfig)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	// Seed with basic data if empty
	if err := database.Seed(db); err != nil {
		slog.Error("Failed to seed the database", "error", err)
		os.Exit(1)
	}

	metricDb, err := database.NewMetricConnection(&cfg.MetricDBConfig)
	if err != nil {
		slog.Error("Failed to connect to metric database", "error", err)
		os.Exit(1)
	}

	lockerDb, err := database.NewLockerConnection(&cfg.SchedulerLockerDBConfig)
	if err != nil {
		slog.Error("Failed to connect to scheduler locker database", "error", err)
		os.Exit(1)
	}

	locker, err := gormlock.NewGormLocker(
		lockerDb,
		cfg.SchedulerLockerConfig.Name,
		gormlock.WithCleanInterval(cfg.SchedulerLockerConfig.CleanInterval),
		gormlock.WithTTL(cfg.SchedulerLockerConfig.TTL),
	)
	if err != nil {
		slog.Error("Failed to create locker", "error", err)
		os.Exit(1)
	}

	// Initialize the store
	store := database.NewGormStore(db)
	metricEntryRepo := database.NewMetricEntryRepository(metricDb)

	// Initialize vault for secret storage (optional)
	var vault schema.Vault
	if cfg.VaultEncryptionKey != "" {
		vaultKey, err := hex.DecodeString(cfg.VaultEncryptionKey)
		if err != nil {
			slog.Error("Invalid vault encryption key (must be 64-character hex string)", "error", err)
			os.Exit(1)
		}
		vault, err = database.NewVault(db, vaultKey)
		if err != nil {
			slog.Error("Failed to initialize vault", "error", err)
			os.Exit(1)
		}
		slog.Info("Vault initialized for secret storage")
	} else {
		slog.Warn("Vault encryption key not configured - secret properties will not work")
	}

	// Initialize schema engine for service property validation
	propertyEngine := domain.NewServicePropertyEngine(store, vault)

	// Initialize commanders
	serviceCmd := domain.NewServiceCommander(store, propertyEngine)
	serviceTypeCmd := domain.NewServiceTypeCommander(store)
	serviceGroupCmd := domain.NewServiceGroupCommander(store)
	serviceOptionTypeCmd := domain.NewServiceOptionTypeCommander(store)
	serviceOptionCmd := domain.NewServiceOptionCommander(store)
	participantCmd := domain.NewParticipantCommander(store)
	agentTypeCmd := domain.NewAgentTypeCommander(store)
	jobCmd := domain.NewJobCommander(store, propertyEngine)
	metricEntryCmd := domain.NewMetricEntryCommander(store, metricEntryRepo)
	metricTypeCmd := domain.NewMetricTypeCommander(store, metricEntryRepo)
	agentCmd := domain.NewAgentCommander(store)
	tokenCmd := domain.NewTokenCommander(store)

	// Initialize authenticators
	authenticators := []auth.Authenticator{}

	for _, authType := range cfg.Authenticators {
		switch strings.TrimSpace(authType) {
		case "token":
			tokenAuth := database.NewTokenAuthenticator(store)
			authenticators = append(authenticators, tokenAuth)
			slog.Info("Token authentication enabled")
		case "oauth":
			ctx := context.Background()
			oauthAuth, err := keycloak.NewAuthenticator(ctx, &cfg.OAuthConfig)
			if err != nil {
				slog.Error("Failed to initialize OAuth authenticator", "error", err)
				os.Exit(1)
			}
			authenticators = append(authenticators, oauthAuth)
			slog.Info("OAuth authentication enabled", "issuer", cfg.OAuthConfig.GetIssuer())
		default:
			slog.Warn("Unknown authenticator type in config", "type", authType)
		}
	}

	if len(authenticators) == 0 {
		slog.Warn("No authenticators enabled in configuration. API will be unprotected.")
		// Optionally, you might want to exit or use a no-op authenticator
	}

	ath := auth.NewCompositeAuthenticator(authenticators...)

	athz := auth.NewRuleBasedAuthorizer(authz.Rules)

	// Initialize commanders for service pools
	servicePoolSetCmd := domain.NewServicePoolSetCommander(store)
	servicePoolCmd := domain.NewServicePoolCommander(store)
	servicePoolValueCmd := domain.NewServicePoolValueCommander(store)

	// Initialize handlers
	agentTypeHandler := api.NewAgentTypeHandler(store.AgentTypeRepo(), agentTypeCmd, athz)
	serviceTypeHandler := api.NewServiceTypeHandler(store.ServiceTypeRepo(), serviceTypeCmd, athz, propertyEngine)
	serviceOptionTypeHandler := api.NewServiceOptionTypeHandler(store.ServiceOptionTypeRepo(), serviceOptionTypeCmd, athz)
	serviceOptionHandler := api.NewServiceOptionHandler(store.ServiceOptionRepo(), serviceOptionCmd, athz)
	servicePoolSetHandler := api.NewServicePoolSetHandler(store.ServicePoolSetRepo(), servicePoolSetCmd, athz)
	servicePoolHandler := api.NewServicePoolHandler(store.ServicePoolRepo(), servicePoolCmd, athz)
	servicePoolValueHandler := api.NewServicePoolValueHandler(store.ServicePoolValueRepo(), servicePoolValueCmd, athz)
	participantHandler := api.NewParticipantHandler(store.ParticipantRepo(), participantCmd, athz)
	agentHandler := api.NewAgentHandler(store.AgentRepo(), agentCmd, athz)
	serviceGroupHandler := api.NewServiceGroupHandler(store.ServiceGroupRepo(), serviceGroupCmd, athz)
	serviceHandler := api.NewServiceHandler(store.ServiceRepo(), store.AgentRepo(), store.ServiceGroupRepo(), serviceCmd, athz)
	jobHandler := api.NewJobHandler(store.JobRepo(), jobCmd, athz)
	metricTypeHandler := api.NewMetricTypeHandler(store.MetricTypeRepo(), metricTypeCmd, athz)
	metricEntryHandler := api.NewMetricEntryHandler(metricEntryRepo, store.ServiceRepo(), metricEntryCmd, athz)
	eventSubscriptionCmd := domain.NewEventSubscriptionCommander(store)
	eventHandler := api.NewEventHandler(store.EventRepo(), eventSubscriptionCmd, athz)
	tokenHandler := api.NewTokenHandler(store.TokenRepo(), tokenCmd, store.AgentRepo(), athz)

	// Initialize vault handler only if vault is configured
	var vaultHandler *api.VaultHandler
	if vault != nil {
		vaultHandler = api.NewVaultHandler(vault)
	}

	serverError := make(chan error, 1)

	var server *http.Server
	var healthServer *http.Server
	if cfg.ApiServer {
		server = BuildHttpServer(&cfg, ath, agentTypeHandler, serviceTypeHandler, serviceOptionTypeHandler, serviceOptionHandler, servicePoolSetHandler, servicePoolHandler, servicePoolValueHandler, participantHandler, agentHandler, serviceGroupHandler, serviceHandler, metricTypeHandler, metricEntryHandler, eventHandler, jobHandler, tokenHandler, vaultHandler, logger)
		// Start main API server
		go func() {
			slog.Info("Server starting", "port", cfg.Port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("Failed to start server", "error", err)
				serverError <- err
			}
		}()

		// Start health server in a goroutine
		healthServer = buildHealthServer(&cfg, db, authenticators)
		go func() {
			slog.Info("Health server starting", "port", cfg.HealthPort)
			if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("Failed to start health server", "error", err)
				serverError <- err
			}
		}()
	}

	var wg sync.WaitGroup

	scheduler, err := gocron.NewScheduler(gocron.WithDistributedLocker(locker))
	if err != nil {
		slog.Error("Failed to create scheduler", "error", err)
		serverError <- err
	}

	if cfg.JobMaintenance {
		task := JobMaintenanceTask(&cfg.JobConfig, store, serviceCmd, &wg)
		err := ScheduleWork(task, &scheduler, cfg.JobConfig.Maintenance, "job_maintenance")
		if err != nil {
			slog.Error("Failed to schedule work", "error", err)
			serverError <- err
		}
	}

	if cfg.AgentMaintenance {
		task := DisconnectUnhealthyAgentsTask(&cfg.AgentConfig, store, &wg)
		err := ScheduleWork(task, &scheduler, cfg.AgentConfig.HealthTimeout, "agent_maintenance")
		if err != nil {
			slog.Error("Failed to schedule work", "error", err)
			serverError <- err
		}
	}

	if cfg.JobMaintenance || cfg.AgentMaintenance {
		go func() {
			scheduler.Start()
		}()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverError:
		slog.Error("Server error", "error", err)
		os.Exit(1)
	case <-stop:
		slog.Info("Shutting down server...")
	}

	if server != nil {
		serverCtx, serverStopCtx := context.WithCancel(context.Background())
		go func() {
			shutdownCtx, shutdownStopCtx := context.WithTimeout(serverCtx, cfg.ShutdownTimeout)
			go func() {
				<-shutdownCtx.Done()
				if shutdownCtx.Err() == context.DeadlineExceeded {
					slog.Error("Server shutdown timed out")
				}
			}()
			slog.Debug("HTTP Server shutdown started")
			err := server.Shutdown(shutdownCtx)
			if err != nil {
				slog.Error("Failed to shutdown server", "error", err)
			}
			serverStopCtx()
			shutdownStopCtx()
		}()
		<-serverCtx.Done()
		slog.Debug("HTTP Server shutdown completed")
	}

	if healthServer != nil {
		serverCtx, serverStopCtx := context.WithCancel(context.Background())
		go func() {
			shutdownCtx, shutdownStopCtx := context.WithTimeout(serverCtx, cfg.ShutdownTimeout)
			go func() {
				<-shutdownCtx.Done()
				if shutdownCtx.Err() == context.DeadlineExceeded {
					slog.Error("Health Server shutdown timed out")
				}
			}()
			slog.Debug("HEALTH Server shutdown started")
			err := healthServer.Shutdown(shutdownCtx)
			if err != nil {
				slog.Error("Failed to shutdown health server", "error", err)
			}
			serverStopCtx()
			shutdownStopCtx()
		}()
		<-serverCtx.Done()
		slog.Debug("HEALTH Server shutdown completed")
	}

	wg.Wait()
}

func BuildHttpServer(
	cfg *config.Config,
	ath auth.Authenticator,
	agentTypeHandler *api.AgentTypeHandler,
	serviceTypeHandler *api.ServiceTypeHandler,
	serviceOptionTypeHandler *api.ServiceOptionTypeHandler,
	serviceOptionHandler *api.ServiceOptionHandler,
	servicePoolSetHandler *api.ServicePoolSetHandler,
	servicePoolHandler *api.ServicePoolHandler,
	servicePoolValueHandler *api.ServicePoolValueHandler,
	participantHandler *api.ParticipantHandler,
	agentHandler *api.AgentHandler,
	serviceGroupHandler *api.ServiceGroupHandler,
	serviceHandler *api.ServiceHandler,
	metricTypeHandler *api.MetricTypeHandler,
	metricEntryHandler *api.MetricEntryHandler,
	eventHandler *api.EventHandler,
	jobHandler *api.JobHandler,
	tokenHandler *api.TokenHandler,
	vaultHandler *api.VaultHandler,
	logger *slog.Logger,
) *http.Server {
	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(
		middleware.RequestID,
		middleware.RequestLogger(&logging.SlogFormatter{Logger: logger}),
		middleware.RealIP,
		middleware.Recoverer,
		render.SetContentType(render.ContentTypeJSON),
	)

	authMiddleware := middlewares.Auth(ath)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Route("/agent-types", agentTypeHandler.Routes())
		r.Route("/service-types", serviceTypeHandler.Routes())
		r.Route("/service-option-types", serviceOptionTypeHandler.Routes())
		r.Route("/service-options", serviceOptionHandler.Routes())
		r.Route("/service-pool-sets", servicePoolSetHandler.Routes())
		r.Route("/service-pools", servicePoolHandler.Routes())
		r.Route("/service-pool-values", servicePoolValueHandler.Routes())
		r.Route("/participants", participantHandler.Routes())
		r.Route("/agents", agentHandler.Routes())
		r.Route("/service-groups", serviceGroupHandler.Routes())
		r.Route("/services", serviceHandler.Routes())
		r.Route("/metric-types", metricTypeHandler.Routes())
		r.Route("/metric-entries", metricEntryHandler.Routes())
		r.Route("/events", eventHandler.Routes())
		r.Route("/jobs", jobHandler.Routes())
		r.Route("/tokens", tokenHandler.Routes())
		
		// Register vault routes only if vault is configured
		if vaultHandler != nil {
			r.Route("/vault/secrets", vaultHandler.Routes())
		}
	})

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: r,
	}
}

func buildHealthServer(cfg *config.Config, db *gorm.DB, authenticators []auth.Authenticator) *http.Server {
	// Initialize health checker and handlers
	healthDeps := &health.PrimaryDependencies{
		DB:             db,
		Authenticators: authenticators,
	}
	healthChecker := health.NewHealthChecker(healthDeps)
	healthHandler := health.NewHandler(healthChecker)

	// Setup health router
	healthRouter := chi.NewRouter()
	healthRouter.Use(
		middleware.RequestID,
		middleware.RealIP,
		middleware.Recoverer,
		render.SetContentType(render.ContentTypeJSON),
	)
	healthRouter.Get("/healthz", healthHandler.HealthHandler)
	healthRouter.Get("/ready", healthHandler.ReadinessHandler)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HealthPort),
		Handler: healthRouter,
	}
}

func DisconnectUnhealthyAgentsTask(cfg *config.AgentConfig, store domain.Store, wg *sync.WaitGroup) gocron.Task {
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

func ScheduleWork(task gocron.Task, scheduler *gocron.Scheduler, duration time.Duration, job_name string) error {

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

func JobMaintenanceTask(cfg *config.JobConfig, store domain.Store, serviceCmd domain.ServiceCommander, wg *sync.WaitGroup) gocron.Task {
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
