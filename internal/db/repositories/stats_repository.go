package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepository struct {
	pool *pgxpool.Pool
}

func NewStatsRepository(pool *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{pool: pool}
}

// Get retrieves trading statistics for a specific user
func (r *StatsRepository) Get(ctx context.Context, userID string) (*models.TradingUserStats, error) {
	query := `
		SELECT 
			user_id, total_trades, total_amount, avg_trade_size,
			first_trade_date, last_trade_date, trading_days, risk_score,
			typical_symbols, trading_hours, last_sync_timestamp,
			created_at, updated_at
		FROM trading_user_stats
		WHERE user_id = $1`

	var stats models.TradingUserStats
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&stats.UserID,
		&stats.TotalTrades,
		&stats.TotalAmount,
		&stats.AvgTradeSize,
		&stats.FirstTradeDate,
		&stats.LastTradeDate,
		&stats.TradingDays,
		&stats.RiskScore,
		&stats.TypicalSymbols,
		&stats.TradingHours,
		&stats.LastSyncTimestamp,
		&stats.CreatedAt,
		&stats.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("stats not found for user: %s", userID)
		}
		return nil, err
	}

	return &stats, nil
}

// Upsert inserts or updates trading statistics for a user
func (r *StatsRepository) Upsert(ctx context.Context, stats *models.TradingUserStats) error {
	query := `
		INSERT INTO trading_user_stats (
			user_id, total_trades, total_amount, avg_trade_size,
			first_trade_date, last_trade_date, trading_days, risk_score,
			typical_symbols, trading_hours, last_sync_timestamp,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (user_id) DO UPDATE SET
			total_trades = EXCLUDED.total_trades,
			total_amount = EXCLUDED.total_amount,
			avg_trade_size = EXCLUDED.avg_trade_size,
			first_trade_date = COALESCE(trading_user_stats.first_trade_date, EXCLUDED.first_trade_date),
			last_trade_date = GREATEST(trading_user_stats.last_trade_date, EXCLUDED.last_trade_date),
			trading_days = EXCLUDED.trading_days,
			risk_score = EXCLUDED.risk_score,
			typical_symbols = EXCLUDED.typical_symbols,
			trading_hours = EXCLUDED.trading_hours,
			last_sync_timestamp = EXCLUDED.last_sync_timestamp,
			updated_at = NOW()`

	now := time.Now()
	if stats.CreatedAt.IsZero() {
		stats.CreatedAt = now
	}
	stats.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		stats.UserID,
		stats.TotalTrades,
		stats.TotalAmount,
		stats.AvgTradeSize,
		stats.FirstTradeDate,
		stats.LastTradeDate,
		stats.TradingDays,
		stats.RiskScore,
		stats.TypicalSymbols,
		stats.TradingHours,
		stats.LastSyncTimestamp,
		stats.CreatedAt,
		stats.UpdatedAt,
	)

	return err
}

// GetLastSyncTime returns the last sync timestamp for a user
func (r *StatsRepository) GetLastSyncTime(ctx context.Context, userID string) (time.Time, error) {
	query := `SELECT last_sync_timestamp FROM trading_user_stats WHERE user_id = $1`

	var lastSync time.Time
	err := r.pool.QueryRow(ctx, query, userID).Scan(&lastSync)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return epoch if no record exists
			return time.Unix(0, 0), nil
		}
		return time.Time{}, err
	}

	return lastSync, nil
}

// GetActiveUsers returns user IDs who have traded within the specified number of days
func (r *StatsRepository) GetActiveUsers(ctx context.Context, days int) ([]string, error) {
	query := `
		SELECT user_id 
		FROM trading_user_stats 
		WHERE last_trade_date >= NOW() - INTERVAL '1 day' * $1
		ORDER BY last_trade_date DESC`

	rows, err := r.pool.Query(ctx, query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}

// UpdateDelta updates stats by adding delta values (for incremental sync)
func (r *StatsRepository) UpdateDelta(ctx context.Context, userID string, newTrades int64, newAmount float64, latestTimestamp time.Time) error {
	query := `
		INSERT INTO trading_user_stats (
			user_id, total_trades, total_amount, avg_trade_size,
			last_trade_date, last_sync_timestamp,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			total_trades = trading_user_stats.total_trades + $2,
			total_amount = trading_user_stats.total_amount + $3,
			avg_trade_size = (trading_user_stats.total_amount + $3) / NULLIF(trading_user_stats.total_trades + $2, 0),
			last_trade_date = GREATEST(trading_user_stats.last_trade_date, $5),
			last_sync_timestamp = $6,
			updated_at = NOW()`

	avgTradeSize := float64(0)
	if newTrades > 0 {
		avgTradeSize = newAmount / float64(newTrades)
	}

	_, err := r.pool.Exec(ctx, query,
		userID,
		newTrades,
		newAmount,
		avgTradeSize,
		latestTimestamp,
		latestTimestamp,
	)

	return err
}

// UpdateSymbolsAndHours updates the typical_symbols and trading_hours JSONB fields
func (r *StatsRepository) UpdateSymbolsAndHours(ctx context.Context, userID string, symbols models.SymbolFrequency, hours models.HourFrequency) error {
	symbolsJSON, err := json.Marshal(symbols)
	if err != nil {
		return fmt.Errorf("failed to marshal symbols: %w", err)
	}

	hoursJSON, err := json.Marshal(hours)
	if err != nil {
		return fmt.Errorf("failed to marshal hours: %w", err)
	}

	query := `
		UPDATE trading_user_stats 
		SET 
			typical_symbols = $2,
			trading_hours = $3,
			updated_at = NOW()
		WHERE user_id = $1`

	_, err = r.pool.Exec(ctx, query, userID, symbolsJSON, hoursJSON)
	return err
}

// UpdateRiskScore updates only the risk score for a user
func (r *StatsRepository) UpdateRiskScore(ctx context.Context, userID string, riskScore float64) error {
	query := `
		UPDATE trading_user_stats 
		SET 
			risk_score = $2,
			updated_at = NOW()
		WHERE user_id = $1`

	result, err := r.pool.Exec(ctx, query, userID, riskScore)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no stats found for user: %s", userID)
	}

	return nil
}
