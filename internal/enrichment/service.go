package enrichment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/enrichment/provider"
	"yeti/internal/logger"
	"yeti/pkg/cel"
	"yeti/pkg/metrics"
	"yeti/pkg/models"
	"yeti/pkg/tracing"
)

func getPayloadKeys(payload map[string]interface{}) []string {
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	return keys
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getEnrichmentRuleIDs(rules []Rule) []string {
	ids := make([]string, 0, len(rules))
	for _, r := range rules {
		ids = append(ids, r.ID)
	}
	return ids
}

func getEnrichmentRuleNames(rules []Rule) []string {
	names := make([]string, 0, len(rules))
	for _, r := range rules {
		names = append(names, r.Name)
	}
	return names
}

func getProviderNames(providers map[string]provider.DataProvider) []string {
	names := make([]string, 0, len(providers))
	for k := range providers {
		names = append(names, k)
	}
	return names
}

type Service interface {
	Process(ctx context.Context, msg models.MessageEnvelope) (models.MessageEnvelope, error)

	ReloadRules(ctx context.Context) error
}

type serviceImpl struct {
	repo      Repository
	cache     *redis.Client
	providers map[string]provider.DataProvider
	evaluator *cel.Evaluator
	rules     []Rule
	rulesMu   sync.RWMutex
	logger    logger.Logger
}

func NewService(repo Repository, cache *redis.Client, log logger.Logger) Service {
	return NewServiceWithCircuitBreaker(repo, cache, log, nil)
}

func NewServiceWithCircuitBreaker(repo Repository, cache *redis.Client, log logger.Logger, cbConfig *config.CircuitBreakerConfig) Service {
	evaluator, err := cel.NewEvaluator()
	if err != nil {
		log.WarnwCtx(context.Background(), "Failed to create CEL evaluator", "error", err)
	}

	s := &serviceImpl{
		repo:      repo,
		cache:     cache,
		providers: make(map[string]provider.DataProvider),
		evaluator: evaluator,
		rules:     make([]Rule, 0),
		logger:    log,
	}

	var apiProv provider.DataProvider = provider.NewAPIProvider()
	if cbConfig != nil {
		apiProv = provider.WrapWithCircuitBreaker(apiProv, "api", *cbConfig)
	}
	s.providers["api"] = apiProv

	if cache != nil {
		var cacheProv provider.DataProvider = provider.NewCacheProvider(cache)
		if cbConfig != nil {
			cacheProv = provider.WrapWithCircuitBreaker(cacheProv, "cache", *cbConfig)
		}
		s.providers["cache"] = cacheProv
		s.providers["redis"] = cacheProv
	}

	return s
}

func NewServiceWithDatabaseProviders(repo Repository, cache *redis.Client, mongoClient *mongo.Client, postgresDB *sql.DB, log logger.Logger) Service {
	return NewServiceWithDatabaseProvidersAndCircuitBreaker(repo, cache, mongoClient, postgresDB, log, nil)
}

func NewServiceWithDatabaseProvidersAndCircuitBreaker(repo Repository, cache *redis.Client, mongoClient *mongo.Client, postgresDB *sql.DB, log logger.Logger, cbConfig *config.CircuitBreakerConfig) Service {
	evaluator, err := cel.NewEvaluator()
	if err != nil {
		log.WarnwCtx(context.Background(), "Failed to create CEL evaluator", "error", err)
	}

	s := &serviceImpl{
		repo:      repo,
		cache:     cache,
		providers: make(map[string]provider.DataProvider),
		evaluator: evaluator,
		rules:     make([]Rule, 0),
		logger:    log,
	}

	var apiProv provider.DataProvider = provider.NewAPIProvider()
	if cbConfig != nil {
		apiProv = provider.WrapWithCircuitBreaker(apiProv, "api", *cbConfig)
	}
	s.providers["api"] = apiProv

	if cache != nil {
		var cacheProv provider.DataProvider = provider.NewCacheProvider(cache)
		if cbConfig != nil {
			cacheProv = provider.WrapWithCircuitBreaker(cacheProv, "cache", *cbConfig)
		}
		s.providers["cache"] = cacheProv
		s.providers["redis"] = cacheProv
		s.logger.InfowCtx(context.Background(), "Cache provider registered")
	}

	if mongoClient != nil {
		var mongoProv provider.DataProvider = provider.NewMongoDBProvider(mongoClient)
		if cbConfig != nil {
			mongoProv = provider.WrapWithCircuitBreaker(mongoProv, "mongodb", *cbConfig)
		}
		s.providers["mongodb"] = mongoProv
		s.logger.InfowCtx(context.Background(), "MongoDB provider registered")
	}

	if postgresDB != nil {
		var pgProv provider.DataProvider = provider.NewPostgreSQLProvider(postgresDB)
		if cbConfig != nil {
			pgProv = provider.WrapWithCircuitBreaker(pgProv, "postgresql", *cbConfig)
		}
		s.providers["postgresql"] = pgProv
		s.logger.InfowCtx(context.Background(), "PostgreSQL provider registered")
	}

	return s
}

func (s *serviceImpl) ReloadRules(ctx context.Context) error {
	rules, err := s.repo.GetActiveRules(ctx)
	if err != nil {
		return err
	}

	s.rulesMu.Lock()
	s.rules = rules
	s.rulesMu.Unlock()

	metrics.SetEnrichmentActiveRules(len(rules))

	s.logger.InfowCtx(ctx, "Reloaded enrichment rules",
		"rules_count", len(rules),
	)
	return nil
}

func (s *serviceImpl) Process(ctx context.Context, msg models.MessageEnvelope) (models.MessageEnvelope, error) {
	ctx, span := tracing.GetTracer("enrichment-service").Start(ctx, "enrichment.process")
	defer span.End()

	s.logger.DebugwCtx(ctx, "Processing message for enrichment",
		"message_id", msg.ID,
		"source", msg.Source,
		"payload_keys", getPayloadKeys(msg.Payload),
	)

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		metrics.ObserveEnrichmentDuration(duration, "success")
	}()

	activeRules := s.getActiveRules()
	s.logger.DebugwCtx(ctx, "Active enrichment rules loaded",
		"rules_count", len(activeRules),
		"rule_ids", getEnrichmentRuleIDs(activeRules),
		"rule_names", getEnrichmentRuleNames(activeRules),
	)

	if msg.Metadata.Enrichment == nil {
		msg.Metadata.Enrichment = make(map[string]interface{})
	}

	var cacheHits, cacheMisses int

	for i, rule := range activeRules {
		if err := ctx.Err(); err != nil {
			return msg, err
		}

		s.logger.DebugwCtx(ctx, "Processing enrichment rule",
			"rule_index", i+1,
			"total_rules", len(activeRules),
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"field_to_enrich", rule.FieldToEnrich,
			"source_type", rule.SourceType,
			"priority", rule.Priority,
			"enabled", rule.Enabled,
		)

		fieldValue, exists := msg.GetPayloadField(rule.FieldToEnrich)
		if !exists {
			s.logger.DebugwCtx(ctx, "Field not found in payload, skipping rule",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"field_to_enrich", rule.FieldToEnrich,
			)
			continue
		}

		s.logger.DebugwCtx(ctx, "Field found in payload",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"field_to_enrich", rule.FieldToEnrich,
			"field_value", fieldValue,
		)

		sourceData, hit, err := s.fetchSourceData(ctx, rule, fieldValue)
		if err != nil {
			if rule.ErrorHandling == constants.ErrorHandlingFail {
				metrics.ObserveEnrichmentDuration(time.Since(start), "error")
				return msg, err
			}
			if IsSkipRuleError(err) {
				s.logger.DebugwCtx(ctx, "Skipping rule due to error handling",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"error_handling", rule.ErrorHandling,
					"error", err,
				)
			}
			continue
		}

		if hit {
			cacheHits++
			s.logger.DebugwCtx(ctx, "Source data retrieved from cache",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"cache_hit", true,
			)
		} else {
			cacheMisses++
			s.logger.DebugwCtx(ctx, "Source data fetched from provider",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"cache_hit", false,
				"source_data_keys", getMapKeys(sourceData),
			)
		}

		s.applyTransformations(ctx, rule, sourceData, &msg)
	}

	s.updateCacheMetrics(cacheHits, cacheMisses)
	
	s.logger.DebugwCtx(ctx, "Enrichment processing completed",
		"message_id", msg.ID,
		"rules_processed", len(activeRules),
		"cache_hits", cacheHits,
		"cache_misses", cacheMisses,
		"enrichment_fields_count", len(msg.Metadata.Enrichment),
		"enrichment_fields", getMapKeys(msg.Metadata.Enrichment),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return msg, nil
}

func (s *serviceImpl) getActiveRules() []Rule {
	s.rulesMu.RLock()
	defer s.rulesMu.RUnlock()

	rules := make([]Rule, len(s.rules))
	copy(rules, s.rules)
	return rules
}

func (s *serviceImpl) fetchSourceData(ctx context.Context, rule Rule, fieldValue interface{}) (map[string]interface{}, bool, error) {
	cacheKey := fmt.Sprintf("%s%s:%v", constants.CacheKeyPrefixEnrich, rule.ID, fieldValue)

	s.logger.DebugwCtx(ctx, "Checking cache for source data",
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"cache_key", cacheKey,
		"field_value", fieldValue,
	)

	val, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		s.logger.DebugwCtx(ctx, "Cache hit, unmarshaling source data",
			"rule_id", rule.ID,
			"cache_key", cacheKey,
		)
		var sourceData map[string]interface{}
		if err := json.Unmarshal([]byte(val), &sourceData); err != nil {
			s.logger.WarnwCtx(ctx, "Failed to unmarshal cache value",
				"error", err,
				"cache_key", cacheKey,
			)
			return nil, false, err
		}
		metrics.EnrichmentMessagesTotal.WithLabelValues("cache_hit").Inc()
		s.logger.DebugwCtx(ctx, "Source data retrieved from cache",
			"rule_id", rule.ID,
			"source_data_keys", getMapKeys(sourceData),
		)
		return sourceData, true, nil
	}

	s.logger.DebugwCtx(ctx, "Cache miss, fetching from provider",
		"rule_id", rule.ID,
		"cache_key", cacheKey,
		"cache_error", err,
	)

	metrics.EnrichmentMessagesTotal.WithLabelValues("cache_miss").Inc()
	return s.fetchFromProvider(ctx, rule, fieldValue, cacheKey)
}

func (s *serviceImpl) fetchFromProvider(ctx context.Context, rule Rule, fieldValue interface{}, cacheKey string) (map[string]interface{}, bool, error) {
	providerName := s.resolveProviderName(rule.SourceType)
	s.logger.DebugwCtx(ctx, "Resolved provider name",
		"rule_id", rule.ID,
		"source_type", rule.SourceType,
		"provider_name", providerName,
	)

	provider, ok := s.providers[providerName]
	if !ok {
		s.logger.ErrorwCtx(ctx, "Provider not registered",
			"rule_id", rule.ID,
			"source_type", rule.SourceType,
			"provider_name", providerName,
			"available_providers", getProviderNames(s.providers),
		)
		return nil, false, fmt.Errorf("unknown source type: %s (provider not registered)", rule.SourceType)
	}

	providerConfig := convertSourceConfig(rule.SourceConfig)
	s.logger.DebugwCtx(ctx, "Fetching data from provider",
		"rule_id", rule.ID,
		"provider_name", providerName,
		"field_value", fieldValue,
		"provider_config", providerConfig,
	)

	fetched, err := provider.Fetch(ctx, providerConfig, fieldValue)
	if err != nil {
		s.logger.DebugwCtx(ctx, "Provider fetch failed",
			"rule_id", rule.ID,
			"provider_name", providerName,
			"error", err,
		)
		return s.handleFetchError(ctx, rule, providerName, err)
	}

	s.logger.DebugwCtx(ctx, "Data fetched from provider",
		"rule_id", rule.ID,
		"provider_name", providerName,
		"fetched_data_keys", getMapKeys(fetched),
	)

	s.cacheSourceData(ctx, cacheKey, fetched, rule.CacheTTLSeconds)
	return fetched, false, nil
}

func (s *serviceImpl) resolveProviderName(sourceType string) string {
	if sourceType == constants.SourceTypeDatabase {
		return constants.ProviderNameMongoDB
	}
	if sourceType == constants.SourceTypeRedis {
		return constants.ProviderNameCache
	}
	return sourceType
}

func (s *serviceImpl) handleFetchError(ctx context.Context, rule Rule, providerName string, err error) (map[string]interface{}, bool, error) {
	metrics.EnrichmentMessagesTotal.WithLabelValues("error").Inc()

	if rule.FallbackValue != nil {
		metrics.FallbackUsageTotal.WithLabelValues("enrichment", "fallback_value", err.Error()).Inc()
		s.logger.WarnwCtx(ctx, "Enrichment failed, using fallback value",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"field_to_enrich", rule.FieldToEnrich,
			"provider", providerName,
			"error", err,
		)
		return map[string]interface{}{"value": rule.FallbackValue}, false, nil
	}

	if rule.ErrorHandling == constants.ErrorHandlingFail {
		return nil, false, fmt.Errorf("enrichment failed for rule %s (field: %s, provider: %s): %w", rule.Name, rule.FieldToEnrich, providerName, err)
	}

	metrics.FallbackUsageTotal.WithLabelValues("enrichment", rule.ErrorHandling, err.Error()).Inc()
	s.logger.WarnwCtx(ctx, "Enrichment failed, skipping rule",
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"field_to_enrich", rule.FieldToEnrich,
		"provider", providerName,
		"error_handling", rule.ErrorHandling,
		"error", err,
	)

	return nil, false, &skipRuleError{rule: rule.Name, reason: err.Error()}
}

func (s *serviceImpl) cacheSourceData(ctx context.Context, cacheKey string, sourceData map[string]interface{}, ttlSeconds int) {
	s.logger.DebugwCtx(ctx, "Caching source data",
		"cache_key", cacheKey,
		"ttl_seconds", ttlSeconds,
		"source_data_keys", getMapKeys(sourceData),
	)

	bytes, err := json.Marshal(sourceData)
	if err != nil {
		s.logger.WarnwCtx(ctx, "Failed to marshal source data",
			"error", err,
			"cache_key", cacheKey,
		)
		return
	}
	if err := s.cache.Set(ctx, cacheKey, bytes, time.Duration(ttlSeconds)*time.Second).Err(); err != nil {
		s.logger.WarnwCtx(ctx, "Failed to cache enrichment data",
			"error", err,
			"cache_key", cacheKey,
		)
	} else {
		s.logger.DebugwCtx(ctx, "Source data cached successfully",
			"cache_key", cacheKey,
			"ttl_seconds", ttlSeconds,
		)
	}
}

func (s *serviceImpl) applyTransformations(ctx context.Context, rule Rule, sourceData map[string]interface{}, msg *models.MessageEnvelope) {
	s.logger.DebugwCtx(ctx, "Applying transformations",
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"transformations_count", len(rule.Transformations),
	)

	for i, trans := range rule.Transformations {
		s.logger.DebugwCtx(ctx, "Processing transformation",
			"rule_id", rule.ID,
			"transformation_index", i+1,
			"total_transformations", len(rule.Transformations),
			"target_field", trans.TargetField,
			"source_path", trans.SourcePath,
			"expression", trans.Expression,
			"has_default", trans.Default != nil,
		)

		fieldValue, exists := s.getSourceFieldValue(trans.SourcePath, sourceData)

		if !exists {
			s.logger.DebugwCtx(ctx, "Source field not found",
				"rule_id", rule.ID,
				"target_field", trans.TargetField,
				"source_path", trans.SourcePath,
			)
			if trans.Default != nil {
				msg.Metadata.Enrichment[trans.TargetField] = trans.Default
				s.logger.DebugwCtx(ctx, "Using default value",
					"rule_id", rule.ID,
					"target_field", trans.TargetField,
					"default_value", trans.Default,
				)
			}
			if trans.Default == nil && rule.ErrorHandling == constants.ErrorHandlingSkipField {
				s.logger.DebugwCtx(ctx, "Skipping field (field not found, skip_field)",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"target_field", trans.TargetField,
					"source_path", trans.SourcePath,
				)
			}
			continue
		}

		s.logger.DebugwCtx(ctx, "Source field found",
			"rule_id", rule.ID,
			"target_field", trans.TargetField,
			"source_path", trans.SourcePath,
			"field_value", fieldValue,
		)

		transformedValue, err := s.transformValue(ctx, fieldValue, trans, rule.Name, *msg, sourceData)
		if err != nil {
			s.logger.DebugwCtx(ctx, "Transformation error",
				"rule_id", rule.ID,
				"target_field", trans.TargetField,
				"expression", trans.Expression,
				"error", err,
			)
			if rule.ErrorHandling == constants.ErrorHandlingFail {
				s.logger.ErrorwCtx(ctx, "Transformation failed",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"target_field", trans.TargetField,
					"error", err,
				)
				return
			} else if rule.ErrorHandling == constants.ErrorHandlingSkipField {
				s.logger.DebugwCtx(ctx, "Skipping field (transformation error, skip_field)",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"target_field", trans.TargetField,
					"error", err,
				)
				if trans.Default != nil {
					msg.Metadata.Enrichment[trans.TargetField] = trans.Default
				}
				continue
			} else {
				s.logger.DebugwCtx(ctx, "Skipping remaining transformations (transformation error, skip_rule)",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"error", err,
				)
				return
			}
		}

		if transformedValue != nil {
			msg.Metadata.Enrichment[trans.TargetField] = transformedValue
			s.logger.DebugwCtx(ctx, "Transformation applied successfully",
				"rule_id", rule.ID,
				"target_field", trans.TargetField,
				"transformed_value", transformedValue,
			)
		} else {
			s.logger.DebugwCtx(ctx, "Transformation returned nil",
				"rule_id", rule.ID,
				"target_field", trans.TargetField,
			)
		}
	}

	s.logger.DebugwCtx(ctx, "All transformations applied",
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"enrichment_fields_added", len(msg.Metadata.Enrichment),
	)
}

func (s *serviceImpl) getSourceFieldValue(sourcePath string, sourceData map[string]interface{}) (interface{}, bool) {
	if sourcePath == "." {
		return sourceData, true
	}
	value, exists := sourceData[sourcePath]
	return value, exists
}

func (s *serviceImpl) transformValue(ctx context.Context, fieldValue interface{}, trans Transformation, ruleName string, msg models.MessageEnvelope, sourceData map[string]interface{}) (interface{}, error) {
	if trans.Expression == "" {
		return fieldValue, nil
	}

	if s.evaluator == nil {
		evaluator, err := cel.NewEvaluator()
		if err != nil {
			return nil, fmt.Errorf("failed to create CEL evaluator: %w", err)
		}
		s.evaluator = evaluator
	}

	transformed, err := s.evaluator.EvaluateTransform(ctx, trans.Expression, msg, sourceData)
	if err != nil {
		return nil, fmt.Errorf("CEL transformation failed: %w", err)
	}
	return transformed, nil
}

func (s *serviceImpl) updateCacheMetrics(cacheHits, cacheMisses int) {
	totalCacheRequests := cacheHits + cacheMisses
	if totalCacheRequests > 0 {
		hitRate := float64(cacheHits) / float64(totalCacheRequests)
		metrics.SetEnrichmentCacheHitRate(hitRate)
	}
}

func convertSourceConfig(cfg SourceConfig) provider.SourceConfig {
	var query *provider.Query
	if cfg.Query != nil {
		query = provider.QueryFromMap(cfg.Query)
	}

	return provider.SourceConfig{
		URL:        cfg.URL,
		Method:     cfg.Method,
		Headers:    cfg.Headers,
		TimeoutMs:  cfg.TimeoutMs,
		RetryCount: cfg.RetryCount,
		Database:   cfg.Database,
		Collection: cfg.Collection,
		Query:      query,
		Field:      cfg.Field,
		KeyPattern: cfg.KeyPattern,
		CacheType:  cfg.CacheType,
	}
}
