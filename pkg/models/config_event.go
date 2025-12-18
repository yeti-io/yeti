package models

import "time"

type ConfigUpdateEvent struct {
	EventType   string                 `json:"event_type"`   // "filtering_rule_updated", "enrichment_rule_updated", "dedup_config_updated"
	ServiceType string                 `json:"service_type"` // "filtering", "enrichment", "deduplication"
	RuleID      string                 `json:"rule_id,omitempty"`
	Action      string                 `json:"action"` // "create", "update", "delete", "toggle"
	Timestamp   time.Time              `json:"timestamp"`
	ChangedBy   string                 `json:"changed_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

const (
	EventTypeFilteringRuleUpdated  = "filtering_rule_updated"
	EventTypeEnrichmentRuleUpdated = "enrichment_rule_updated"
	EventTypeDedupConfigUpdated    = "dedup_config_updated"
)

const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionToggle = "toggle"
	ActionReload = "reload"
)

const (
	ServiceTypeFiltering     = "filtering"
	ServiceTypeEnrichment    = "enrichment"
	ServiceTypeDeduplication = "deduplication"
)
