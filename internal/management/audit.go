package management

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AuditLogger struct {
	db *sql.DB
}

func NewAuditLogger(db *sql.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

func (a *AuditLogger) LogRuleChange(ctx context.Context, entry AuditLogEntry) error {
	query := `
		INSERT INTO rule_audit_logs (id, rule_id, rule_type, action, old_value, new_value, changed_by, change_reason, ip_address, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	id := uuid.New().String()
	if entry.ID != "" {
		id = entry.ID
	}

	oldValueJSON, _ := json.Marshal(entry.OldValue)
	newValueJSON, _ := json.Marshal(entry.NewValue)

	var ruleID *string
	if entry.RuleID != "" {
		ruleID = &entry.RuleID
	}

	var ipAddress *string
	if entry.IPAddress != "" {
		ipAddress = &entry.IPAddress
	}

	var changeReason *string
	if entry.ChangeReason != "" {
		changeReason = &entry.ChangeReason
	}

	timestamp := time.Now()
	if !entry.Timestamp.IsZero() {
		timestamp = entry.Timestamp
	}

	_, err := a.db.ExecContext(ctx, query,
		id, ruleID, entry.RuleType, entry.Action,
		oldValueJSON, newValueJSON,
		entry.ChangedBy, changeReason, ipAddress, timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to log audit entry: %w", err)
	}

	return nil
}

type AuditLogEntry struct {
	ID           string
	RuleID       string
	RuleType     string
	Action       string
	OldValue     interface{}
	NewValue     interface{}
	ChangedBy    string
	ChangeReason string
	IPAddress    string
	Timestamp    time.Time
}
