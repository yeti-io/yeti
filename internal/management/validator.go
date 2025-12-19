package management

import (
	"fmt"

	"yeti/pkg/cel"
)

func ValidateFilteringRule(req CreateFilteringRuleRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Expression == "" {
		return fmt.Errorf("expression is required")
	}

	evaluator, err := cel.NewEvaluator()
	if err != nil {
		return fmt.Errorf("failed to create CEL evaluator: %w", err)
	}

	if err := evaluator.ValidateFilterExpression(req.Expression); err != nil {
		return fmt.Errorf("invalid CEL expression: %w", err)
	}

	return nil
}

func ValidateUpdateFilteringRule(req UpdateFilteringRuleRequest) error {
	if req.Expression != nil && *req.Expression != "" {
		evaluator, err := cel.NewEvaluator()
		if err != nil {
			return fmt.Errorf("failed to create CEL evaluator: %w", err)
		}

		if err := evaluator.ValidateFilterExpression(*req.Expression); err != nil {
			return fmt.Errorf("invalid CEL expression: %w", err)
		}
	}
	return nil
}

var validSourceTypes = map[string]bool{
	"api":        true,
	"database":   true,
	"mongodb":    true,
	"postgresql": true,
	"cache":      true,
	"redis":      true,
}

var validErrorHandling = map[string]bool{
	"skip_field": true,
	"skip_rule":  true,
	"fail":       true,
}

func ValidateEnrichmentRule(req CreateEnrichmentRuleRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.FieldToEnrich == "" {
		return fmt.Errorf("field_to_enrich is required")
	}
	if !validSourceTypes[req.SourceType] {
		return fmt.Errorf("invalid source_type: %s. Allowed: api, database, mongodb, postgresql, cache, redis", req.SourceType)
	}
	if req.SourceType == "api" && req.SourceConfig.URL == "" {
		return fmt.Errorf("source_config.url is required for api source type")
	}
	if req.SourceType == "database" || req.SourceType == "mongodb" || req.SourceType == "postgresql" {
		if req.SourceConfig.Collection == "" {
			return fmt.Errorf("source_config.collection is required for database source type")
		}
		if req.SourceConfig.Query == nil && req.SourceConfig.Field == "" {
			return fmt.Errorf("either source_config.query or source_config.field is required for database source type")
		}
	}
	if req.SourceType == "cache" || req.SourceType == "redis" {
		if req.SourceConfig.KeyPattern == "" {
			return fmt.Errorf("source_config.key_pattern is required for cache source type")
		}
	}
	if req.ErrorHandling != "" && !validErrorHandling[req.ErrorHandling] {
		return fmt.Errorf("invalid error_handling: %s. Allowed: skip_field, skip_rule, fail", req.ErrorHandling)
	}
	if req.CacheTTLSeconds < 0 {
		return fmt.Errorf("cache_ttl_seconds must be non-negative")
	}

	evaluator, err := cel.NewEvaluator()
	if err != nil {
		return fmt.Errorf("failed to create CEL evaluator: %w", err)
	}

	for i, trans := range req.Transformations {
		if trans.Expression != "" {
			if err := evaluator.ValidateTransformExpression(trans.Expression); err != nil {
				return fmt.Errorf("invalid CEL expression in transformation[%d]: %w", i, err)
			}
		}
	}

	return nil
}

func ValidateUpdateEnrichmentRule(req UpdateEnrichmentRuleRequest) error {
	if req.SourceType != nil {
		if !validSourceTypes[*req.SourceType] {
			return fmt.Errorf("invalid source_type: %s. Allowed: api, database, cache", *req.SourceType)
		}
	}
	if req.ErrorHandling != nil {
		if !validErrorHandling[*req.ErrorHandling] {
			return fmt.Errorf("invalid error_handling: %s. Allowed: skip_field, skip_rule, fail", *req.ErrorHandling)
		}
	}
	if req.CacheTTLSeconds != nil && *req.CacheTTLSeconds < 0 {
		return fmt.Errorf("cache_ttl_seconds must be non-negative")
	}

	if req.Transformations != nil {
		evaluator, err := cel.NewEvaluator()
		if err != nil {
			return fmt.Errorf("failed to create CEL evaluator: %w", err)
		}

		for i, trans := range *req.Transformations {
			if trans.Expression != "" {
				if err := evaluator.ValidateExpression(trans.Expression); err != nil {
					return fmt.Errorf("invalid CEL expression in transformation[%d]: %w", i, err)
				}
			}
		}
	}

	return nil
}

var validHashAlgorithms = map[string]bool{
	"md5":    true,
	"sha256": true,
}

var validOnRedisError = map[string]bool{
	"allow":      true,
	"filter_out": true,
}

func ValidateDeduplicationConfig(req UpdateDeduplicationConfigRequest) error {
	if req.HashAlgorithm != nil {
		if !validHashAlgorithms[*req.HashAlgorithm] {
			return fmt.Errorf("invalid hash_algorithm: %s. Allowed: md5, sha256", *req.HashAlgorithm)
		}
	}
	if req.OnRedisError != nil {
		if !validOnRedisError[*req.OnRedisError] {
			return fmt.Errorf("invalid on_redis_error: %s. Allowed: allow, filter_out", *req.OnRedisError)
		}
	}
	if req.TTLSeconds != nil && *req.TTLSeconds <= 0 {
		return fmt.Errorf("ttl_seconds must be positive")
	}
	if req.FieldsToHash != nil {
		if len(*req.FieldsToHash) == 0 {
			return fmt.Errorf("fields_to_hash cannot be empty")
		}
	}
	return nil
}
