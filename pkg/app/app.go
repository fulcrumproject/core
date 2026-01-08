package app

import (
	"context"
	"encoding/hex"
	"flag"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/config"
	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/gormlock"
	"github.com/fulcrumproject/core/pkg/health"
	"github.com/fulcrumproject/core/pkg/keycloak"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/utils/confbuilder"
	"github.com/fulcrumproject/utils/logging"
	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"
)

type App struct {
	Config                   *config.Config
	Db                       *gorm.DB
	MetricDb                 *gorm.DB
	Authenticators           []auth.Authenticator
	AgentTypeHandler         *api.AgentTypeHandler
	ServiceTypeHandler       *api.ServiceTypeHandler
	ServiceOptionTypeHandler *api.ServiceOptionTypeHandler
	ServiceOptionHandler     *api.ServiceOptionHandler
	ServicePoolSetHandler    *api.ServicePoolSetHandler
	ServicePoolHandler       *api.ServicePoolHandler
	ServicePoolValueHandler  *api.ServicePoolValueHandler
	ParticipantHandler       *api.ParticipantHandler
	AgentHandler             *api.AgentHandler
	ServiceGroupHandler      *api.ServiceGroupHandler
	ServiceHandler           *api.ServiceHandler
	MetricTypeHandler        *api.MetricTypeHandler
	MetricEntryHandler       *api.MetricEntryHandler
	MetricEntryRepo          *database.GormMetricEntryRepository
	EventHandler             *api.EventHandler
	JobHandler               *api.JobHandler
	TokenHandler             *api.TokenHandler
	VaultHandler             *api.VaultHandler
	HealthHandler            *health.Handler
	Logger                   *slog.Logger
	PropertyEngine           *schema.Engine[domain.ServicePropertyContext]
	CompositeAuthenticator   *auth.CompositeAuthenticator
	RuleBasedAuthorizer      *authz.RuleBasedAuthorizer
	Store                    domain.Store
	ServiceCmd               domain.ServiceCommander
	Scheduler                *gocron.Scheduler
	scheduleStarted          bool
	WaitGroup                *sync.WaitGroup
}

func readConfig() (*config.Config, error) {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	cfg, err := confbuilder.New(config.Default).
		EnvPrefix(config.EnvPrefix).
		EnvFiles(".env").
		File(configPath).
		Build()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func initLogger(cfg *config.Config) *slog.Logger {
	logger := logging.NewLogger(&cfg.LogConfig)
	slog.SetDefault(logger)

	slog.Debug("API_SERVER", "value", cfg.ApiServer)
	slog.Debug("JOB_MAINTENANCE", "value", cfg.JobMaintenance)
	slog.Debug("AGENT_MAINTENANCE", "value", cfg.AgentMaintenance)

	return logger
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	db, err := database.NewConnection(&cfg.DBConfig)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initMetricDatabase(cfg *config.Config) (*gorm.DB, error) {
	db, err := database.NewMetricConnection(&cfg.MetricDBConfig)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initLockerDatabase(cfg *config.Config) (*gorm.DB, error) {
	db, err := database.NewLockerConnection(&cfg.SchedulerLockerDBConfig)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initScheduler(cfg *config.Config, lockerDb *gorm.DB) (*gocron.Scheduler, error) {
	locker, err := gormlock.NewGormLocker(
		lockerDb,
		cfg.SchedulerLockerConfig.Name,
		gormlock.WithCleanInterval(cfg.SchedulerLockerConfig.CleanInterval),
		gormlock.WithTTL(cfg.SchedulerLockerConfig.TTL),
	)
	if err != nil {
		slog.Error("Failed to create locker", "error", err)
		return nil, err
	}
	scheduler, err := gocron.NewScheduler(gocron.WithDistributedLocker(locker))
	if err != nil {
		slog.Error("Failed to create scheduler", "error", err)
		return nil, err
	}
	return &scheduler, nil
}

func NewApp() *App {
	cfg, err := readConfig()
	if err != nil {
		slog.Error("Invalid configuration", "error", err)
		return nil
	}

	logger := initLogger(cfg)

	db, err := initDatabase(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		return nil
	}

	// Seed the database with initial data
	if err := database.Seed(db); err != nil {
		slog.Error("Failed to seed database", "error", err)
		return nil
	}

	metricDb, err := initMetricDatabase(cfg)
	if err != nil {
		slog.Error("Failed to initialize metric database", "error", err)
		return nil
	}

	lockerDb, err := initLockerDatabase(cfg)
	if err != nil {
		slog.Error("Failed to initialize locker database", "error", err)
		return nil
	}

	scheduler, err := initScheduler(cfg, lockerDb)
	if err != nil {
		slog.Error("Failed to initialize scheduler", "error", err)
		return nil
	}

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
	propertyEngine := domain.NewServicePropertyEngine(vault)

	// Initialize schema engine for agent configuration validation
	agentConfigEngine := domain.NewAgentConfigSchemaEngine(vault)

	serviceCmd := domain.NewServiceCommander(store, propertyEngine)
	serviceTypeCmd := domain.NewServiceTypeCommander(store, propertyEngine)
	serviceGroupCmd := domain.NewServiceGroupCommander(store)
	serviceOptionTypeCmd := domain.NewServiceOptionTypeCommander(store)
	serviceOptionCmd := domain.NewServiceOptionCommander(store)
	participantCmd := domain.NewParticipantCommander(store)
	agentTypeCmd := domain.NewAgentTypeCommander(store, agentConfigEngine)
	jobCmd := domain.NewJobCommander(store, propertyEngine)
	metricEntryCmd := domain.NewMetricEntryCommander(store, metricEntryRepo)
	metricTypeCmd := domain.NewMetricTypeCommander(store, metricEntryRepo)
	agentCmd := domain.NewAgentCommander(store, agentConfigEngine)
	tokenCmd := domain.NewTokenCommander(store)
	eventSubscriptionCmd := domain.NewEventSubscriptionCommander(store)

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

	athz := authz.NewRuleBasedAuthorizer(authz.Rules)

	// Initialize commanders for service pools
	servicePoolSetCmd := domain.NewServicePoolSetCommander(store)
	servicePoolCmd := domain.NewServicePoolCommander(store)
	servicePoolValueCmd := domain.NewServicePoolValueCommander(store)

	return &App{
		Config:                   cfg,
		Db:                       db,
		MetricDb:                 metricDb,
		Logger:                   logger,
		Scheduler:                scheduler,
		scheduleStarted:          false,
		WaitGroup:                &sync.WaitGroup{},
		Store:                    store,
		Authenticators:           authenticators,
		CompositeAuthenticator:   ath,
		RuleBasedAuthorizer:      athz,
		ServiceTypeHandler:       api.NewServiceTypeHandler(store.ServiceTypeRepo(), serviceTypeCmd, athz, propertyEngine),
		ServiceOptionTypeHandler: api.NewServiceOptionTypeHandler(store.ServiceOptionTypeRepo(), serviceOptionTypeCmd, athz),
		ServiceOptionHandler:     api.NewServiceOptionHandler(store.ServiceOptionRepo(), serviceOptionCmd, athz),
		ServicePoolSetHandler:    api.NewServicePoolSetHandler(store.ServicePoolSetRepo(), servicePoolSetCmd, athz),
		ServicePoolHandler:       api.NewServicePoolHandler(store.ServicePoolRepo(), servicePoolCmd, athz),
		ServicePoolValueHandler:  api.NewServicePoolValueHandler(store.ServicePoolValueRepo(), servicePoolValueCmd, athz),
		ParticipantHandler:       api.NewParticipantHandler(store.ParticipantRepo(), participantCmd, athz),
		AgentHandler:             api.NewAgentHandler(store.AgentRepo(), agentCmd, athz),
		AgentTypeHandler:         api.NewAgentTypeHandler(store.AgentTypeRepo(), agentTypeCmd, athz),
		ServiceGroupHandler:      api.NewServiceGroupHandler(store.ServiceGroupRepo(), serviceGroupCmd, athz),
		ServiceHandler:           api.NewServiceHandler(store.ServiceRepo(), store.AgentRepo(), store.ServiceGroupRepo(), serviceCmd, athz),
		JobHandler:               api.NewJobHandler(store.JobRepo(), jobCmd, athz),
		MetricTypeHandler:        api.NewMetricTypeHandler(store.MetricTypeRepo(), metricTypeCmd, athz),
		MetricEntryHandler:       api.NewMetricEntryHandler(metricEntryRepo, store.ServiceRepo(), metricEntryCmd, athz),
		MetricEntryRepo:          metricEntryRepo,
		EventHandler:             api.NewEventHandler(store.EventRepo(), eventSubscriptionCmd, athz),
		TokenHandler:             api.NewTokenHandler(store.TokenRepo(), tokenCmd, store.AgentRepo(), athz),
		VaultHandler:             api.NewVaultHandler(vault),
		ServiceCmd:               serviceCmd,
		PropertyEngine:           propertyEngine,
	}
}

func (a *App) StartScheduler() {
	if a.scheduleStarted {
		return
	}
	(*a.Scheduler).Start()
	a.scheduleStarted = true
}
