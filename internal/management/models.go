package management

import "time"

type FilteringRule struct {
	ID         string    `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Expression string    `json:"expression" db:"expression"`
	Priority   int       `json:"priority" db:"priority"`
	Enabled    bool      `json:"enabled" db:"enabled"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type CreateFilteringRuleRequest struct {
	Name       string `json:"name" binding:"required"`
	Expression string `json:"expression" binding:"required"`
	Priority   int    `json:"priority"`
	Enabled    *bool  `json:"enabled"`
}

type UpdateFilteringRuleRequest struct {
	Name       *string `json:"name"`
	Expression *string `json:"expression"`
	Priority   *int    `json:"priority"`
	Enabled    *bool   `json:"enabled"`
}

type EnrichmentRule struct {
	ID              string                     `json:"id" bson:"_id,omitempty"`
	Name            string                     `json:"name" bson:"name"`
	FieldToEnrich   string                     `json:"field_to_enrich" bson:"field_to_enrich"`
	SourceType      string                     `json:"source_type" bson:"source_type"`
	SourceConfig    EnrichmentSourceConfig     `json:"source_config" bson:"source_config"`
	Transformations []EnrichmentTransformation `json:"transformations" bson:"transformations"`
	CacheTTLSeconds int                        `json:"cache_ttl_seconds" bson:"cache_ttl_seconds"`
	ErrorHandling   string                     `json:"error_handling" bson:"error_handling"`
	FallbackValue   interface{}                `json:"fallback_value,omitempty" bson:"fallback_value"`
	Priority        int                        `json:"priority" bson:"priority"`
	Enabled         bool                       `json:"enabled" bson:"enabled"`
	CreatedAt       time.Time                  `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at" bson:"updated_at"`
}

type EnrichmentSourceConfig struct {
	URL        string            `json:"url,omitempty" bson:"url"`
	Method     string            `json:"method,omitempty" bson:"method"`
	Headers    map[string]string `json:"headers,omitempty" bson:"headers"`
	TimeoutMs  int               `json:"timeout_ms,omitempty" bson:"timeout_ms"`
	RetryCount int               `json:"retry_count,omitempty" bson:"retry_count"`

	Database   string                 `json:"database,omitempty" bson:"database"`
	Collection string                 `json:"collection,omitempty" bson:"collection"`
	Query      map[string]interface{} `json:"query,omitempty" bson:"query"`
	Field      string                 `json:"field,omitempty" bson:"field"`

	KeyPattern string `json:"key_pattern,omitempty" bson:"key_pattern"`
	CacheType  string `json:"cache_type,omitempty" bson:"cache_type"`
}

type EnrichmentTransformation struct {
	SourcePath  string      `json:"source_path" bson:"source_path"`
	TargetField string      `json:"target_field" bson:"target_field"`
	Expression  string      `json:"expression,omitempty" bson:"expression"`
	Default     interface{} `json:"default,omitempty" bson:"default"`
}

type CreateEnrichmentRuleRequest struct {
	Name            string                     `json:"name" binding:"required"`
	FieldToEnrich   string                     `json:"field_to_enrich" binding:"required"`
	SourceType      string                     `json:"source_type" binding:"required"`
	SourceConfig    EnrichmentSourceConfig     `json:"source_config" binding:"required"`
	Transformations []EnrichmentTransformation `json:"transformations"`
	CacheTTLSeconds int                        `json:"cache_ttl_seconds"`
	ErrorHandling   string                     `json:"error_handling"`
	FallbackValue   interface{}                `json:"fallback_value"`
	Priority        int                        `json:"priority"`
	Enabled         *bool                      `json:"enabled"`
}

type UpdateEnrichmentRuleRequest struct {
	Name            *string                     `json:"name"`
	FieldToEnrich   *string                     `json:"field_to_enrich"`
	SourceType      *string                     `json:"source_type"`
	SourceConfig    *EnrichmentSourceConfig     `json:"source_config"`
	Transformations *[]EnrichmentTransformation `json:"transformations"`
	CacheTTLSeconds *int                        `json:"cache_ttl_seconds"`
	ErrorHandling   *string                     `json:"error_handling"`
	FallbackValue   *interface{}                `json:"fallback_value"`
	Priority        *int                        `json:"priority"`
	Enabled         *bool                       `json:"enabled"`
}

type DeduplicationConfig struct {
	HashAlgorithm string   `json:"hash_algorithm"`
	TTLSeconds    int      `json:"ttl_seconds"`
	OnRedisError  string   `json:"on_redis_error"`
	FieldsToHash  []string `json:"fields_to_hash"`
}

type UpdateDeduplicationConfigRequest struct {
	HashAlgorithm *string   `json:"hash_algorithm,omitempty"`
	TTLSeconds    *int      `json:"ttl_seconds,omitempty"`
	OnRedisError  *string   `json:"on_redis_error,omitempty"`
	FieldsToHash  *[]string `json:"fields_to_hash,omitempty"`
}
