package app

import (
	"context"
	"log/slog"

	"github.com/gauravdhanuka4/trade-detection-system/config"
	"github.com/gauravdhanuka4/trade-detection-system/internal/alert"
	"github.com/gauravdhanuka4/trade-detection-system/internal/db"
	"github.com/gauravdhanuka4/trade-detection-system/internal/redis"
	"github.com/gauravdhanuka4/trade-detection-system/internal/rules"
	"github.com/gauravdhanuka4/trade-detection-system/internal/utils"
)

// App holds all shared application dependencies
type App struct {
	Config       *config.Config
	Logger       *slog.Logger
	Database     *db.DB
	Redis        redis.RedisClient
	RuleEngine   *rules.RuleEngine
	AlertService alert.Service
}

// New creates and initializes a new application context
// serviceName is used for structured logging context (e.g., "worker", "api")
func New(ctx context.Context, serviceName string) (*App, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	// Initialize structured logger with service context
	logger := utils.NewLogger(utils.LoggerConfig{
		Level:       cfg.GetLogLevel(),
		Format:      "json",
		AddSource:   cfg.IsDevelopment(),
		ServiceName: serviceName,
	})

	logger.Info("Initializing application",
		"environment", cfg.App.Environment,
		"log_level", cfg.GetLogLevel(),
	)

	// Initialize database connection
	logger.Info("Connecting to database...")
	database, err := db.New(cfg)
	if err != nil {
		utils.Fatal(logger, "Failed to connect to database", "error", err)
		return nil, err
	}
	logger.Info("Database connection established",
		"host", cfg.Database.Host,
		"database", cfg.Database.Database,
	)

	// Initialize Redis client
	logger.Info("Connecting to Redis...")
	redisClient, err := redis.NewRedisClient(cfg.Redis)
	if err != nil {
		database.Close()
		utils.Fatal(logger, "Failed to connect to Redis", "error", err)
		return nil, err
	}
	logger.Info("Redis connection established",
		"host", cfg.Redis.Host,
		"db", cfg.Redis.DB,
	)

	// Initialize rule engine (loads rules from database)
	logger.Info("Initializing rule engine...")
	ruleEngine := rules.NewRuleEngine(ctx, database, redisClient, logger)

	// Initialize alert service
	logger.Info("Initializing alert service...")
	alertService := alert.NewAlertService(database.Alerts, database.Trades)

	logger.Info("Application initialized successfully",
		"service", serviceName,
		"enabled_rules", ruleEngine.GetEnabledRulesCount(),
	)

	return &App{
		Config:       cfg,
		Logger:       logger,
		Database:     database,
		Redis:        redisClient,
		RuleEngine:   ruleEngine,
		AlertService: alertService,
	}, nil
}

// Close gracefully shuts down all application resources
func (a *App) Close() {
	a.Logger.Info("Shutting down application...")

	if a.Redis != nil {
		a.Logger.Info("Closing Redis connection...")
		a.Redis.Close()
	}

	if a.Database != nil {
		a.Logger.Info("Closing database connection...")
		a.Database.Close()
	}

	a.Logger.Info("Application shutdown complete")
}

// NewInstance creates a new App or panics if initialization fails
// Use this in main() for simpler error handling
func NewInstance(ctx context.Context, serviceName string) *App {
	app, err := New(ctx, serviceName)
	if err != nil {
		panic(err)
	}
	return app
}
