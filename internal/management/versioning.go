package management

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RuleVersion struct {
	ID           string    `json:"id"`
	RuleID       string    `json:"rule_id"`
	RuleType     string    `json:"rule_type"`
	RuleData     string    `json:"rule_data"`
	Version      int       `json:"version"`
	ChangedBy    string    `json:"changed_by,omitempty"`
	ChangeReason string    `json:"change_reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditLog struct {
	ID           string                 `json:"id"`
	RuleID       *string                `json:"rule_id,omitempty"`
	RuleType     string                 `json:"rule_type"`
	Action       string                 `json:"action"`
	OldValue     map[string]interface{} `json:"old_value,omitempty"`
	NewValue     map[string]interface{} `json:"new_value,omitempty"`
	ChangedBy    string                 `json:"changed_by"`
	ChangeReason string                 `json:"change_reason,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

type VersioningRepository interface {
	CreateVersion(ctx context.Context, version *RuleVersion) error
	GetVersions(ctx context.Context, ruleID string) ([]RuleVersion, error)
	GetVersion(ctx context.Context, ruleID string, version int) (*RuleVersion, error)
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	GetAuditLogs(ctx context.Context, ruleID *string, ruleType string, limit int) ([]AuditLog, error)
	GetNextVersion(ctx context.Context, ruleID string) (int, error)
}

type postgresVersioningRepository struct {
	db *sql.DB
}

func (r *postgresVersioningRepository) CreateVersion(ctx context.Context, version *RuleVersion) error {
	if version.ID == "" {
		version.ID = uuid.New().String()
	}
	if version.CreatedAt.IsZero() {
		version.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO rule_versions (id, rule_id, rule_type, rule_data, version, changed_by, change_reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		version.ID, version.RuleID, version.RuleType, version.RuleData,
		version.Version, version.ChangedBy, version.ChangeReason, version.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create rule version: %w", err)
	}

	return nil
}

func (r *postgresVersioningRepository) GetVersions(ctx context.Context, ruleID string) ([]RuleVersion, error) {
	query := `
		SELECT id, rule_id, rule_type, rule_data, version, changed_by, change_reason, created_at
		FROM rule_versions
		WHERE rule_id = $1
		ORDER BY version DESC
	`

	rows, err := r.db.QueryContext(ctx, query, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	var versions []RuleVersion
	for rows.Next() {
		var v RuleVersion
		if err := rows.Scan(
			&v.ID, &v.RuleID, &v.RuleType, &v.RuleData,
			&v.Version, &v.ChangedBy, &v.ChangeReason, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, nil
}

func (r *postgresVersioningRepository) GetVersion(ctx context.Context, ruleID string, version int) (*RuleVersion, error) {
	query := `
		SELECT id, rule_id, rule_type, rule_data, version, changed_by, change_reason, created_at
		FROM rule_versions
		WHERE rule_id = $1 AND version = $2
	`

	var v RuleVersion
	err := r.db.QueryRowContext(ctx, query, ruleID, version).Scan(
		&v.ID, &v.RuleID, &v.RuleType, &v.RuleData,
		&v.Version, &v.ChangedBy, &v.ChangeReason, &v.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	return &v, nil
}

func (r *postgresVersioningRepository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	var oldValueJSON, newValueJSON []byte
	var err error

	if log.OldValue != nil {
		oldValueJSON, err = json.Marshal(log.OldValue)
		if err != nil {
			return fmt.Errorf("failed to marshal old value: %w", err)
		}
	}

	if log.NewValue != nil {
		newValueJSON, err = json.Marshal(log.NewValue)
		if err != nil {
			return fmt.Errorf("failed to marshal new value: %w", err)
		}
	}

	query := `
		INSERT INTO rule_audit_logs (id, rule_id, rule_type, action, old_value, new_value, changed_by, change_reason, ip_address, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.ExecContext(ctx, query,
		log.ID, log.RuleID, log.RuleType, log.Action,
		oldValueJSON, newValueJSON, log.ChangedBy, log.ChangeReason, log.IPAddress, log.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (r *postgresVersioningRepository) GetAuditLogs(ctx context.Context, ruleID *string, ruleType string, limit int) ([]AuditLog, error) {
	var query string
	var args []interface{}

	if ruleID != nil {
		query = `
			SELECT id, rule_id, rule_type, action, old_value, new_value, changed_by, change_reason, ip_address, timestamp
			FROM rule_audit_logs
			WHERE rule_id = $1
			ORDER BY timestamp DESC
			LIMIT $2
		`
		args = []interface{}{*ruleID, limit}
	} else if ruleType != "" {
		query = `
			SELECT id, rule_id, rule_type, action, old_value, new_value, changed_by, change_reason, ip_address, timestamp
			FROM rule_audit_logs
			WHERE rule_type = $1
			ORDER BY timestamp DESC
			LIMIT $2
		`
		args = []interface{}{ruleType, limit}
	} else {
		query = `
			SELECT id, rule_id, rule_type, action, old_value, new_value, changed_by, change_reason, ip_address, timestamp
			FROM rule_audit_logs
			ORDER BY timestamp DESC
			LIMIT $1
		`
		args = []interface{}{limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var oldValueJSON, newValueJSON []byte
		var ruleIDPtr *string

		if err := rows.Scan(
			&log.ID, &ruleIDPtr, &log.RuleType, &log.Action,
			&oldValueJSON, &newValueJSON, &log.ChangedBy, &log.ChangeReason, &log.IPAddress, &log.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		log.RuleID = ruleIDPtr

		if len(oldValueJSON) > 0 {
			if err := json.Unmarshal(oldValueJSON, &log.OldValue); err != nil {
				return nil, fmt.Errorf("failed to unmarshal old value: %w", err)
			}
		}

		if len(newValueJSON) > 0 {
			if err := json.Unmarshal(newValueJSON, &log.NewValue); err != nil {
				return nil, fmt.Errorf("failed to unmarshal new value: %w", err)
			}
		}

		logs = append(logs, log)
	}

	return logs, nil
}

func ruleToJSON(rule *FilteringRule) (string, error) {
	data := map[string]interface{}{
		"id":         rule.ID,
		"name":       rule.Name,
		"expression": rule.Expression,
		"priority":   rule.Priority,
		"enabled":    rule.Enabled,
		"created_at": rule.CreatedAt,
		"updated_at": rule.UpdatedAt,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func (r *postgresVersioningRepository) GetNextVersion(ctx context.Context, ruleID string) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) + 1 FROM rule_versions WHERE rule_id = $1`

	var version int
	err := r.db.QueryRowContext(ctx, query, ruleID).Scan(&version)
	if err != nil {
		return 1, nil // First version
	}

	return version, nil
}

func NewVersioningRepository(db *sql.DB) VersioningRepository {
	return &postgresVersioningRepository{db: db}
}
