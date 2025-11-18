package management

import (
	"context"
)

type Service interface {
	CreateFilteringRule(ctx context.Context, req CreateFilteringRuleRequest) (*FilteringRule, error)
	ListFilteringRules(ctx context.Context) ([]FilteringRule, error)
	GetFilteringRule(ctx context.Context, id string) (*FilteringRule, error)
	UpdateFilteringRule(ctx context.Context, id string, req UpdateFilteringRuleRequest) (*FilteringRule, error)
	DeleteFilteringRule(ctx context.Context, id string) error
	GetRuleVersions(ctx context.Context, ruleID string) ([]RuleVersion, error)
	GetAuditLogs(ctx context.Context, ruleID *string, ruleType string, limit int) ([]AuditLog, error)

	CreateEnrichmentRule(ctx context.Context, req CreateEnrichmentRuleRequest) (*EnrichmentRule, error)
	ListEnrichmentRules(ctx context.Context) ([]EnrichmentRule, error)
	GetEnrichmentRule(ctx context.Context, id string) (*EnrichmentRule, error)
	UpdateEnrichmentRule(ctx context.Context, id string, req UpdateEnrichmentRuleRequest) (*EnrichmentRule, error)
	DeleteEnrichmentRule(ctx context.Context, id string) error

	GetDeduplicationConfig(ctx context.Context) (*DeduplicationConfig, error)
	UpdateDeduplicationConfig(ctx context.Context, req UpdateDeduplicationConfigRequest) (*DeduplicationConfig, error)
}
