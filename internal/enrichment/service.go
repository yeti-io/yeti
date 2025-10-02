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

type Service interface {
	Process(ctx context.Context, msg models.MessageEnvelope) (models.MessageEnvelope, error)

	ReloadRules(ctx context.Context, skipJitter ...bool) error
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

func (s *serviceImpl) ReloadRules(ctx context.Context, skipJitter ...bool) error {
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

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		metrics.ObserveEnrichmentDuration(duration, "success")
	}()

	activeRules := s.getActiveRules()
	if msg.Metadata.Enrichment == nil {
		msg.Metadata.Enrichment = make(map[string]interface{})
	}

	var cacheHits, cacheMisses int

	for _, rule := range activeRules {
		if err := ctx.Err(); err != nil {
			return msg, err
		}

		fieldValue, exists := msg.GetPayloadField(rule.FieldToEnrich)
		if !exists {
			continue
		}

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
		} else {
			cacheMisses++
		}

		s.applyTransformations(ctx, rule, sourceData, &msg)
	}

	s.updateCacheMetrics(cacheHits, cacheMisses)
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

	val, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var sourceData map[string]interface{}
		if err := json.Unmarshal([]byte(val), &sourceData); err != nil {
			s.logger.WarnwCtx(ctx, "Failed to unmarshal cache value",
				"error", err,
				"cache_key", cacheKey,
			)
			return nil, false, err
		}
		metrics.EnrichmentMessagesTotal.WithLabelValues("cache_hit").Inc()
		return sourceData, true, nil
	}

	metrics.EnrichmentMessagesTotal.WithLabelValues("cache_miss").Inc()
	return s.fetchFromProvider(ctx, rule, fieldValue, cacheKey)
}

func (s *serviceImpl) fetchFromProvider(ctx context.Context, rule Rule, fieldValue interface{}, cacheKey string) (map[string]interface{}, bool, error) {
	providerName := s.resolveProviderName(rule.SourceType)
	provider, ok := s.providers[providerName]
	if !ok {
		return nil, false, fmt.Errorf("unknown source type: %s (provider not registered)", rule.SourceType)
	}

	providerConfig := convertSourceConfig(rule.SourceConfig)
	fetched, err := provider.Fetch(ctx, providerConfig, fieldValue)
	if err != nil {
		return s.handleFetchError(ctx, rule, providerName, err)
	}

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
	}
}

func (s *serviceImpl) applyTransformations(ctx context.Context, rule Rule, sourceData map[string]interface{}, msg *models.MessageEnvelope) {
	for _, trans := range rule.Transformations {
		fieldValue, exists := s.getSourceFieldValue(trans.SourcePath, sourceData)

		if !exists {
			if trans.Default != nil {
				msg.Metadata.Enrichment[trans.TargetField] = trans.Default
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

		transformedValue, err := s.transformValue(ctx, fieldValue, trans, rule.Name, *msg, sourceData)
		if err != nil {
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
		}
	}
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
