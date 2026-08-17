package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AlertRepository struct {
	pool *pgxpool.Pool
}

func NewAlertRepository(pool *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{pool: pool}
}

func (r *AlertRepository) Create(ctx context.Context, alert *models.Alert) error {
	query := `
		INSERT INTO alerts (id, trade_id, rule_id, risk_type, severity, status, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		alert.ID,
		alert.TradeID,
		alert.RuleID,
		alert.RiskType,
		alert.Severity,
		alert.Status,
		alert.Description,
		alert.CreatedAt,
		alert.UpdatedAt,
	)

	return err
}

func (r *AlertRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Alert, error) {
	query := `
		SELECT id, trade_id, rule_id, risk_type, severity, status, description, created_at, updated_at
		FROM alerts
		WHERE id = $1`

	var alert models.Alert
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&alert.ID,
		&alert.TradeID,
		&alert.RuleID,
		&alert.RiskType,
		&alert.Severity,
		&alert.Status,
		&alert.Description,
		&alert.CreatedAt,
		&alert.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert not found")
		}
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) GetByTradeID(ctx context.Context, tradeID uuid.UUID) ([]*models.Alert, error) {
	query := `
		SELECT id, trade_id, rule_id, risk_type, severity, status, description, created_at, updated_at
		FROM alerts
		WHERE trade_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, tradeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(
			&alert.ID,
			&alert.TradeID,
			&alert.RuleID,
			&alert.RiskType,
			&alert.Severity,
			&alert.Status,
			&alert.Description,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, rows.Err()
}

func (r *AlertRepository) GetByStatus(ctx context.Context, status models.AlertStatus, limit, offset int) ([]*models.Alert, error) {
	query := `
		SELECT id, trade_id, rule_id, risk_type, severity, status, description, created_at, updated_at
		FROM alerts
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(
			&alert.ID,
			&alert.TradeID,
			&alert.RuleID,
			&alert.RiskType,
			&alert.Severity,
			&alert.Status,
			&alert.Description,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, rows.Err()
}

func (r *AlertRepository) GetBySeverity(ctx context.Context, severity models.SeverityType, limit, offset int) ([]*models.Alert, error) {
	query := `
		SELECT id, trade_id, rule_id, risk_type, severity, status, description, created_at, updated_at
		FROM alerts
		WHERE severity = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, severity, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(
			&alert.ID,
			&alert.TradeID,
			&alert.RuleID,
			&alert.RiskType,
			&alert.Severity,
			&alert.Status,
			&alert.Description,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, rows.Err()
}

func (r *AlertRepository) Update(ctx context.Context, alert *models.Alert) error {
	query := `
		UPDATE alerts
		SET status = $2, description = $3, updated_at = $4
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		alert.ID,
		alert.Status,
		alert.Description,
		alert.UpdatedAt,
	)

	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("alert not found")
	}

	return nil
}

func (r *AlertRepository) BatchCreate(ctx context.Context, alerts []*models.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	// Build batch insert query
	valueStrings := make([]string, 0, len(alerts))
	valueArgs := make([]interface{}, 0, len(alerts)*9)

	for i, alert := range alerts {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*9+1, i*9+2, i*9+3, i*9+4, i*9+5, i*9+6, i*9+7, i*9+8, i*9+9))

		valueArgs = append(valueArgs,
			alert.ID,
			alert.TradeID,
			alert.RuleID,
			alert.RiskType,
			alert.Severity,
			alert.Status,
			alert.Description,
			alert.CreatedAt,
			alert.UpdatedAt,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO alerts (id, trade_id, rule_id, risk_type, severity, status, description, created_at, updated_at)
		VALUES %s`, strings.Join(valueStrings, ","))

	_, err := r.pool.Exec(ctx, query, valueArgs...)
	return err
}

func (r *AlertRepository) List(ctx context.Context, limit, offset int) ([]*models.Alert, error) {
	query := `
		SELECT id, trade_id, rule_id, risk_type, severity, status, description, created_at, updated_at
		FROM alerts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		var alert models.Alert
		err := rows.Scan(
			&alert.ID,
			&alert.TradeID,
			&alert.RuleID,
			&alert.RiskType,
			&alert.Severity,
			&alert.Status,
			&alert.Description,
			&alert.CreatedAt,
			&alert.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, &alert)
	}

	return alerts, rows.Err()
}
