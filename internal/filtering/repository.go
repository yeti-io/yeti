package filtering

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository interface {
	GetActiveRules(ctx context.Context) ([]Rule, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetActiveRules(ctx context.Context) ([]Rule, error) {
	query := `
		SELECT id, name, expression, priority, enabled, created_at, updated_at
		FROM filtering_rules
		WHERE enabled = true
		ORDER BY priority DESC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rule Rule
		if err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Expression,
			&rule.Priority,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return rules, nil
}
