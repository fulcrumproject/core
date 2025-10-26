package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/fulcrumproject/core/pkg/health"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/utils/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type ApiServer struct {
	app          *App
	server       *http.Server
	healthServer *http.Server
}

func NewApiServer(app *App) *ApiServer {
	return &ApiServer{
		app:          app,
		server:       buildHttpServer(app),
		healthServer: buildHealthServer(app),
	}
}

func (a *ApiServer) Start() error {
	serverError := make(chan error, 1)
	go func() {
		slog.Info("Server starting", "port", a.app.Config.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start server", "error", err)
			serverError <- err
		}

	}()

	go func() {
		slog.Info("Health server starting", "port", a.app.Config.HealthPort)
		if err := a.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start health server", "error", err)
			serverError <- err
		}
	}()

	close(serverError)

	err, open := <-serverError
	if open {
		return err
	}
	return nil
}

func (a *ApiServer) Close() {
	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	go func() {
		shutdownCtx, shutdownStopCtx := context.WithTimeout(serverCtx, a.app.Config.ShutdownTimeout)
		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				slog.Error("Server shutdown timed out")
			}
		}()
		slog.Debug("HTTP Server shutdown started")
		err := a.server.Shutdown(shutdownCtx)
		if err != nil {
			slog.Error("Failed to shutdown server", "error", err)
		}
		serverStopCtx()
		shutdownStopCtx()
	}()
	<-serverCtx.Done()
	slog.Debug("HTTP Server shutdown completed")

	serverCtx, serverStopCtx = context.WithCancel(context.Background())
	go func() {
		shutdownCtx, shutdownStopCtx := context.WithTimeout(serverCtx, a.app.Config.ShutdownTimeout)
		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				slog.Error("Health Server shutdown timed out")
			}
		}()
		slog.Debug("HEALTH Server shutdown started")
		err := a.healthServer.Shutdown(shutdownCtx)
		if err != nil {
			slog.Error("Failed to shutdown health server", "error", err)
		}
		serverStopCtx()
		shutdownStopCtx()
	}()
	<-serverCtx.Done()
	slog.Debug("HEALTH Server shutdown completed")
}

func buildHttpServer(
	app *App,
) *http.Server {
	// Initialize router
	r := chi.NewRouter()

	// Middleware
	r.Use(
		middleware.RequestID,
		middleware.RequestLogger(&logging.SlogFormatter{Logger: app.Logger}),
		middleware.RealIP,
		middleware.Recoverer,
		render.SetContentType(render.ContentTypeJSON),
	)

	authMiddleware := middlewares.Auth(app.CompositeAuthenticator)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Route("/agent-types", app.AgentTypeHandler.Routes())
		r.Route("/service-types", app.ServiceTypeHandler.Routes())
		r.Route("/service-option-types", app.ServiceOptionTypeHandler.Routes())
		r.Route("/service-options", app.ServiceOptionHandler.Routes())
		r.Route("/service-pool-sets", app.ServicePoolSetHandler.Routes())
		r.Route("/service-pools", app.ServicePoolHandler.Routes())
		r.Route("/service-pool-values", app.ServicePoolValueHandler.Routes())
		r.Route("/participants", app.ParticipantHandler.Routes())
		r.Route("/agents", app.AgentHandler.Routes())
		r.Route("/service-groups", app.ServiceGroupHandler.Routes())
		r.Route("/services", app.ServiceHandler.Routes())
		r.Route("/metric-types", app.MetricTypeHandler.Routes())
		r.Route("/metric-entries", app.MetricEntryHandler.Routes())
		r.Route("/events", app.EventHandler.Routes())
		r.Route("/jobs", app.JobHandler.Routes())
		r.Route("/tokens", app.TokenHandler.Routes())
	})

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", app.Config.Port),
		Handler: r,
	}
}

func buildHealthServer(app *App) *http.Server {
	// Initialize health checker and handlers
	healthDeps := &health.PrimaryDependencies{
		DB:             app.Db,
		Authenticators: app.Authenticators,
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
		Addr:    fmt.Sprintf(":%d", app.Config.HealthPort),
		Handler: healthRouter,
	}
}
