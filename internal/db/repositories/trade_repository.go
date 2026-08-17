package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TradeRepository struct {
	pool *pgxpool.Pool
}

func NewTradeRepository(pool *pgxpool.Pool) *TradeRepository {
	return &TradeRepository{pool: pool}
}

func (r *TradeRepository) Create(ctx context.Context, trade *models.Trade) error {
	query := `
		INSERT INTO trades (id, user_id, symbol, amount, price, trade_type, timestamp, source, raw_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.pool.Exec(ctx, query,
		trade.ID,
		trade.UserID,
		trade.Symbol,
		trade.Amount,
		trade.Price,
		trade.Type,
		trade.Timestamp,
		trade.Source,
		trade.RawData,
		trade.CreatedAt,
	)

	return err
}

func (r *TradeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Trade, error) {
	query := `
		SELECT id, user_id, symbol, amount, price, trade_type, timestamp, source, raw_data, created_at
		FROM trades
		WHERE id = $1`

	var trade models.Trade
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&trade.ID,
		&trade.UserID,
		&trade.Symbol,
		&trade.Amount,
		&trade.Price,
		&trade.Type,
		&trade.Timestamp,
		&trade.Source,
		&trade.RawData,
		&trade.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("trade not found")
		}
		return nil, err
	}

	return &trade, nil
}

func (r *TradeRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Trade, error) {
	query := `
		SELECT id, user_id, symbol, amount, price, trade_type, timestamp, source, raw_data, created_at
		FROM trades
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		var trade models.Trade
		err := rows.Scan(
			&trade.ID,
			&trade.UserID,
			&trade.Symbol,
			&trade.Amount,
			&trade.Price,
			&trade.Type,
			&trade.Timestamp,
			&trade.Source,
			&trade.RawData,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		trades = append(trades, &trade)
	}

	return trades, rows.Err()
}

func (r *TradeRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*models.Trade, error) {
	query := `
		SELECT id, user_id, symbol, amount, price, trade_type, timestamp, source, raw_data, created_at
		FROM trades
		WHERE timestamp BETWEEN $1 AND $2
		ORDER BY timestamp DESC`

	rows, err := r.pool.Query(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		var trade models.Trade
		err := rows.Scan(
			&trade.ID,
			&trade.UserID,
			&trade.Symbol,
			&trade.Amount,
			&trade.Price,
			&trade.Type,
			&trade.Timestamp,
			&trade.Source,
			&trade.RawData,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		trades = append(trades, &trade)
	}

	return trades, rows.Err()
}

func (r *TradeRepository) GetBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*models.Trade, error) {
	query := `
		SELECT id, user_id, symbol, amount, price, trade_type, timestamp, source, raw_data, created_at
		FROM trades
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, symbol, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		var trade models.Trade
		err := rows.Scan(
			&trade.ID,
			&trade.UserID,
			&trade.Symbol,
			&trade.Amount,
			&trade.Price,
			&trade.Type,
			&trade.Timestamp,
			&trade.Source,
			&trade.RawData,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		trades = append(trades, &trade)
	}

	return trades, rows.Err()
}

func (r *TradeRepository) BatchCreate(ctx context.Context, trades []*models.Trade) error {
	if len(trades) == 0 {
		return nil
	}

	// Build batch insert query
	valueStrings := make([]string, 0, len(trades))
	valueArgs := make([]interface{}, 0, len(trades)*10)

	for i, trade := range trades {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*10+1, i*10+2, i*10+3, i*10+4, i*10+5, i*10+6, i*10+7, i*10+8, i*10+9, i*10+10))

		valueArgs = append(valueArgs,
			trade.ID,
			trade.UserID,
			trade.Symbol,
			trade.Amount,
			trade.Price,
			trade.Type,
			trade.Timestamp,
			trade.Source,
			trade.RawData,
			trade.CreatedAt,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO trades (id, user_id, symbol, amount, price, trade_type, timestamp, source, raw_data, created_at)
		VALUES %s`, strings.Join(valueStrings, ","))

	_, err := r.pool.Exec(ctx, query, valueArgs...)
	return err
}

// Fraud detection methods
func (r *TradeRepository) GetByUserInTimeWindow(ctx context.Context, userID string, start, end time.Time) ([]*models.Trade, error) {
	query := `
		SELECT id, user_id, symbol, amount, price, trade_type, timestamp, source, raw_data, created_at
		FROM trades
		WHERE user_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp DESC`

	rows, err := r.pool.Query(ctx, query, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		var trade models.Trade
		err := rows.Scan(
			&trade.ID,
			&trade.UserID,
			&trade.Symbol,
			&trade.Amount,
			&trade.Price,
			&trade.Type,
			&trade.Timestamp,
			&trade.Source,
			&trade.RawData,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		trades = append(trades, &trade)
	}

	return trades, rows.Err()
}

func (r *TradeRepository) GetRecentTradesByUser(ctx context.Context, userID string, hours int) ([]*models.Trade, error) {
	query := `
		SELECT id, user_id, symbol, amount, price, trade_type, timestamp, source, raw_data, created_at
		FROM trades
		WHERE user_id = $1 AND timestamp >= NOW() - INTERVAL '%d hours'
		ORDER BY timestamp DESC`

	formattedQuery := fmt.Sprintf(query, hours)
	rows, err := r.pool.Query(ctx, formattedQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*models.Trade
	for rows.Next() {
		var trade models.Trade
		err := rows.Scan(
			&trade.ID,
			&trade.UserID,
			&trade.Symbol,
			&trade.Amount,
			&trade.Price,
			&trade.Type,
			&trade.Timestamp,
			&trade.Source,
			&trade.RawData,
			&trade.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		trades = append(trades, &trade)
	}

	return trades, rows.Err()
}
