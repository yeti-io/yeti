package management

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	pkgerrors "yeti/pkg/errors"
)

type Repository interface {
	CreateFilteringRule(ctx context.Context, rule *FilteringRule) error
	ListFilteringRules(ctx context.Context) ([]FilteringRule, error)
	GetFilteringRule(ctx context.Context, id string) (*FilteringRule, error)
	UpdateFilteringRule(ctx context.Context, rule *FilteringRule) error
	DeleteFilteringRule(ctx context.Context, id string) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateFilteringRule(ctx context.Context, rule *FilteringRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	query := `
		INSERT INTO filtering_rules (id, name, expression, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		rule.ID, rule.Name, rule.Expression,
		rule.Priority, rule.Enabled, rule.CreatedAt, rule.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return pkgerrors.ErrConflict.WithCause(err).WithDetail("message", fmt.Sprintf("rule with name '%s' already exists", rule.Name))
			}
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return pkgerrors.ErrConflict.WithCause(err).WithDetail("message", fmt.Sprintf("rule with name '%s' already exists", rule.Name))
		}
		return fmt.Errorf("failed to create rule: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetFilteringRule(ctx context.Context, id string) (*FilteringRule, error) {
	query := `
		SELECT id, name, expression, priority, enabled, created_at, updated_at
		FROM filtering_rules
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)

	var rule FilteringRule
	err := row.Scan(
		&rule.ID, &rule.Name, &rule.Expression,
		&rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("rule not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	return &rule, nil
}

func (r *PostgresRepository) ListFilteringRules(ctx context.Context) ([]FilteringRule, error) {
	query := `
		SELECT id, name, expression, priority, enabled, created_at, updated_at
		FROM filtering_rules
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer rows.Close()

	var rules []FilteringRule
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		var rule FilteringRule
		if err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Expression,
			&rule.Priority, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (r *PostgresRepository) UpdateFilteringRule(ctx context.Context, rule *FilteringRule) error {
	rule.UpdatedAt = time.Now()

	query := `
		UPDATE filtering_rules
		SET name = $1, expression = $2, priority = $3, enabled = $4, updated_at = $5
		WHERE id = $6
	`

	res, err := r.db.ExecContext(ctx, query,
		rule.Name, rule.Expression,
		rule.Priority, rule.Enabled, rule.UpdatedAt, rule.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}

func (r *PostgresRepository) DeleteFilteringRule(ctx context.Context, id string) error {
	query := `DELETE FROM filtering_rules WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("rule not found")
	}

	return nil
}
