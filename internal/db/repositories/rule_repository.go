package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RuleRepository struct {
	pool *pgxpool.Pool
}

func NewRuleRepository(pool *pgxpool.Pool) *RuleRepository {
	return &RuleRepository{pool: pool}
}

func (r *RuleRepository) Create(ctx context.Context, rule *models.Rule) error {
	query := `
		INSERT INTO rules (id, name, type, conditions, threshold, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, query,
		rule.ID,
		rule.Name,
		rule.Type,
		rule.Conditions,
		rule.Threshold,
		rule.Enabled,
		rule.CreatedAt,
		rule.UpdatedAt,
	)

	return err
}

func (r *RuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Rule, error) {
	query := `
		SELECT id, name, type, conditions, threshold, enabled, created_at, updated_at
		FROM rules
		WHERE id = $1`

	var rule models.Rule
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Type,
		&rule.Conditions,
		&rule.Threshold,
		&rule.Enabled,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rule not found")
		}
		return nil, err
	}

	return &rule, nil
}

func (r *RuleRepository) GetEnabled(ctx context.Context) ([]*models.Rule, error) {
	query := `
		SELECT id, name, type, conditions, threshold, enabled, created_at, updated_at
		FROM rules
		WHERE enabled = true
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.Rule
	for rows.Next() {
		var rule models.Rule
		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Type,
			&rule.Conditions,
			&rule.Threshold,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}

	return rules, rows.Err()
}

func (r *RuleRepository) GetByType(ctx context.Context, riskType models.RiskType) ([]*models.Rule, error) {
	query := `
		SELECT id, name, type, conditions, threshold, enabled, created_at, updated_at
		FROM rules
		WHERE type = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, riskType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.Rule
	for rows.Next() {
		var rule models.Rule
		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Type,
			&rule.Conditions,
			&rule.Threshold,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}

	return rules, rows.Err()
}

func (r *RuleRepository) Update(ctx context.Context, rule *models.Rule) error {
	query := `
		UPDATE rules
		SET name = $2, type = $3, conditions = $4, threshold = $5, enabled = $6, updated_at = $7
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		rule.ID,
		rule.Name,
		rule.Type,
		rule.Conditions,
		rule.Threshold,
		rule.Enabled,
		rule.UpdatedAt,
	)

	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}

func (r *RuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM rules WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}

func (r *RuleRepository) List(ctx context.Context, limit, offset int) ([]*models.Rule, error) {
	query := `
		SELECT id, name, type, conditions, threshold, enabled, created_at, updated_at
		FROM rules
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.Rule
	for rows.Next() {
		var rule models.Rule
		err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Type,
			&rule.Conditions,
			&rule.Threshold,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, &rule)
	}

	return rules, rows.Err()
}
