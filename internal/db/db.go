package db

import (
	"context"
	"fmt"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/config"
	"github.com/gauravdhanuka4/trade-detection-system/internal/db/repositories"
	"github.com/gauravdhanuka4/trade-detection-system/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository interfaces - these define the contracts
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
}

type TradeRepository interface {
	Create(ctx context.Context, trade *models.Trade) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Trade, error)
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Trade, error)
	GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.Trade, error)
	GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*models.Trade, error)
	BatchCreate(ctx context.Context, trades []*models.Trade) error

	// Fraud detection methods
	GetByUserInTimeWindow(ctx context.Context, userID string, start, end time.Time) ([]*models.Trade, error)
	GetRecentTradesByUser(ctx context.Context, userID string, hours int) ([]*models.Trade, error)
}

type StatsRepository interface {
	Get(ctx context.Context, userID string) (*models.TradingUserStats, error)
	Upsert(ctx context.Context, stats *models.TradingUserStats) error
	GetLastSyncTime(ctx context.Context, userID string) (time.Time, error)
	GetActiveUsers(ctx context.Context, days int) ([]string, error)
	UpdateRiskScore(ctx context.Context, userID string, riskScore float64) error
	UpdateDelta(ctx context.Context, userID string, newTrades int64, newAmount float64, lastTradeTime time.Time) error
	UpdateSymbolsAndHours(ctx context.Context, userID string, symbols models.SymbolFrequency, hours models.HourFrequency) error
}

type RuleRepository interface {
	Create(ctx context.Context, rule *models.Rule) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Rule, error)
	GetEnabled(ctx context.Context) ([]*models.Rule, error)
	GetByType(ctx context.Context, riskType models.RiskType) ([]*models.Rule, error)
	Update(ctx context.Context, rule *models.Rule) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.Rule, error)
}

type AlertRepository interface {
	Create(ctx context.Context, alert *models.Alert) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Alert, error)
	GetByTradeID(ctx context.Context, tradeID uuid.UUID) ([]*models.Alert, error)
	GetByStatus(ctx context.Context, status models.AlertStatus, limit, offset int) ([]*models.Alert, error)
	GetBySeverity(ctx context.Context, severity models.SeverityType, limit, offset int) ([]*models.Alert, error)
	Update(ctx context.Context, alert *models.Alert) error
	BatchCreate(ctx context.Context, alerts []*models.Alert) error
	List(ctx context.Context, limit, offset int) ([]*models.Alert, error)
}

// DB represents the database connection and repositories
type DB struct {
	Users  UserRepository
	Trades TradeRepository
	Rules  RuleRepository
	Alerts AlertRepository
	Stats  StatsRepository
}

// New creates a new database connection using config
func New(cfg *config.Config) (*DB, error) {
	// Get DSN from config
	dsn := buildConnectionString(cfg.Database)

	// Create connection pool with default settings
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create DB instance
	db := &DB{}

	// Initialize repository implementations
	db.Users = repositories.NewUserRepository(pool)
	db.Trades = repositories.NewTradeRepository(pool)
	db.Rules = repositories.NewRuleRepository(pool)
	db.Alerts = repositories.NewAlertRepository(pool)
	db.Stats = repositories.NewStatsRepository(pool)

	return db, nil
}

func buildConnectionString(dbCfg models.DatabaseConfig) string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.Database,
		dbCfg.Username,
		dbCfg.Password,
		dbCfg.SSLMode,
	)
}

// Close closes the database connection pool
func (db *DB) Close() {
	for _, repo := range []interface{}{
		db.Users,
		db.Trades,
		db.Rules,
		db.Alerts,
		db.Stats,
	} {
		if closer, ok := repo.(interface{ Close() }); ok {
			closer.Close()
		}
	}
}
