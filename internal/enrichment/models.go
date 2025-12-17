package enrichment

import "time"

type Rule struct {
	ID              string           `bson:"_id,omitempty"`
	Name            string           `bson:"name"`
	FieldToEnrich   string           `bson:"field_to_enrich"`
	SourceType      string           `bson:"source_type"` // api, database, cache
	SourceConfig    SourceConfig     `bson:"source_config"`
	Transformations []Transformation `bson:"transformations"`
	CacheTTLSeconds int              `bson:"cache_ttl_seconds"`
	ErrorHandling   string           `bson:"error_handling"` // skip_field, skip_rule, fail
	FallbackValue   interface{}      `bson:"fallback_value"`
	Priority        int              `bson:"priority"`
	Enabled         bool             `bson:"enabled"`
	CreatedAt       time.Time        `bson:"created_at"`
	UpdatedAt       time.Time        `bson:"updated_at"`
}

type SourceConfig struct {
	URL        string            `bson:"url,omitempty"`
	Method     string            `bson:"method,omitempty"`
	Headers    map[string]string `bson:"headers,omitempty"`
	TimeoutMs  int               `bson:"timeout_ms,omitempty"`
	RetryCount int               `bson:"retry_count,omitempty"`

	Database   string                 `bson:"database,omitempty"`
	Collection string                 `bson:"collection,omitempty"`
	Query      map[string]interface{} `bson:"query,omitempty"`
	Field      string                 `bson:"field,omitempty"`

	KeyPattern string `bson:"key_pattern,omitempty"`
	CacheType  string `bson:"cache_type,omitempty"`
}

type Transformation struct {
	SourcePath  string      `bson:"source_path"`
	TargetField string      `bson:"target_field"`
	Expression  string      `bson:"expression"`
	Default     interface{} `bson:"default"`
}
