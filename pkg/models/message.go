package models

import "time"

type MessageEnvelope struct {
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`  // Business data
	Metadata  Metadata               `json:"metadata"` // Pipeline metadata (trace_id, processing_info)
}

type Metadata struct {
	TraceID        string                 `json:"trace_id,omitempty"`
	FiltersApplied *FiltersApplied        `json:"filters_applied,omitempty"`
	Deduplication  *DeduplicationInfo     `json:"deduplication,omitempty"`
	Enrichment     map[string]interface{} `json:"enrichment,omitempty"`
}

type FiltersApplied struct {
	PassedAt time.Time `json:"passed_at"`
	RuleIDs  []string  `json:"rule_ids"`
}

type DeduplicationInfo struct {
	IsUnique  bool      `json:"is_unique"`
	CheckedAt time.Time `json:"checked_at"`
}
