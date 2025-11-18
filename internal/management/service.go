package management

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"yeti/internal/config"
	"yeti/internal/constants"
	pkgerrors "yeti/pkg/errors"
	"yeti/pkg/models"
)

type service struct {
	repo                Repository
	enrichmentRepo      EnrichmentRepository
	versioningRepo      VersioningRepository
	configEventProducer *ConfigEventProducer
	auditEnabled        bool
	dedupConfig         *DeduplicationConfig
	dedupConfigMu       sync.RWMutex
}

type ServiceOption func(*service)

func WithVersioning(versioningRepo VersioningRepository) ServiceOption {
	return func(s *service) {
		s.versioningRepo = versioningRepo
		s.auditEnabled = true
	}
}

func WithEnrichment(enrichmentRepo EnrichmentRepository) ServiceOption {
	return func(s *service) {
		s.enrichmentRepo = enrichmentRepo
	}
}

func WithConfigEvents(configEventProducer *ConfigEventProducer) ServiceOption {
	return func(s *service) {
		s.configEventProducer = configEventProducer
	}
}

func WithDeduplicationConfig(dedupCfg config.DeduplicationConfig) ServiceOption {
	return func(s *service) {
		fieldsToHash := dedupCfg.FieldsToHash
		if len(fieldsToHash) == 0 {
			fieldsToHash = []string{"id", "source"}
		}

		s.dedupConfig = &DeduplicationConfig{
			HashAlgorithm: dedupCfg.HashAlgorithm,
			TTLSeconds:    dedupCfg.TTLSeconds,
			OnRedisError:  dedupCfg.OnRedisError,
			FieldsToHash:  fieldsToHash,
		}
	}
}

func NewService(repo Repository, opts ...ServiceOption) Service {
	s := &service{
		repo:         repo,
		auditEnabled: false,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.versioningRepo != nil {
		s.auditEnabled = true
	}

	return s
}

func (s *service) CreateFilteringRule(ctx context.Context, req CreateFilteringRuleRequest) (*FilteringRule, error) {
	if err := ValidateFilteringRule(req); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrValidation)
	}

	rule := &FilteringRule{
		Name:       req.Name,
		Expression: req.Expression,
		Priority:   req.Priority,
		Enabled:    getEnabledValue(req.Enabled),
	}

	if err := s.repo.CreateFilteringRule(ctx, rule); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}

	s.createVersionAndAudit(ctx, rule, "create", nil)
	s.publishConfigEvent(ctx, models.ActionCreate, rule.ID)

	return s.copyFilteringRule(rule), nil
}

func (s *service) ListFilteringRules(ctx context.Context) ([]FilteringRule, error) {
	domainRules, err := s.repo.ListFilteringRules(ctx)
	if err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}
	rules := make([]FilteringRule, len(domainRules))
	for i, dr := range domainRules {
		rules[i] = FilteringRule{
			ID:         dr.ID,
			Name:       dr.Name,
			Expression: dr.Expression,
			Priority:   dr.Priority,
			Enabled:    dr.Enabled,
			CreatedAt:  dr.CreatedAt,
			UpdatedAt:  dr.UpdatedAt,
		}
	}
	return rules, nil
}

func (s *service) GetFilteringRule(ctx context.Context, id string) (*FilteringRule, error) {
	rule, err := s.repo.GetFilteringRule(ctx, id)
	if err != nil {
		return nil, s.handleNotFoundError(err, id)
	}
	if rule == nil {
		return nil, pkgerrors.ErrNotFound.WithDetail("id", id)
	}
	return s.copyFilteringRule(rule), nil
}

func (s *service) UpdateFilteringRule(ctx context.Context, id string, req UpdateFilteringRuleRequest) (*FilteringRule, error) {
	if err := ValidateUpdateFilteringRule(req); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrValidation)
	}

	rule, err := s.repo.GetFilteringRule(ctx, id)
	if err != nil {
		return nil, s.handleNotFoundError(err, id)
	}
	if rule == nil {
		return nil, pkgerrors.ErrNotFound.WithDetail("id", id)
	}

	oldValue, _ := s.ruleToMap(rule)
	s.updateFilteringRuleFields(rule, req)

	if err := s.repo.UpdateFilteringRule(ctx, rule); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}

	s.createVersionAndAudit(ctx, rule, "update", oldValue)
	s.publishConfigEvent(ctx, models.ActionUpdate, rule.ID)

	return s.copyFilteringRule(rule), nil
}

func (s *service) DeleteFilteringRule(ctx context.Context, id string) error {
	rule, err := s.repo.GetFilteringRule(ctx, id)
	if err != nil {
		return s.handleNotFoundError(err, id)
	}
	if rule == nil {
		return pkgerrors.ErrNotFound.WithDetail("id", id)
	}

	oldValue, _ := s.ruleToMap(rule)

	if err := s.repo.DeleteFilteringRule(ctx, id); err != nil {
		return pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}

	if s.auditEnabled && s.versioningRepo != nil {
		auditLog := s.buildAuditLog(id, "filtering", "delete", oldValue, nil, getChangedBy(ctx))
		_ = s.versioningRepo.CreateAuditLog(ctx, auditLog)
	}

	s.publishConfigEvent(ctx, models.ActionDelete, id)
	return nil
}

func (s *service) GetRuleVersions(ctx context.Context, ruleID string) ([]RuleVersion, error) {
	if s.versioningRepo == nil {
		return nil, pkgerrors.ErrInternal.WithDetail("message", "versioning not enabled")
	}
	versions, err := s.versioningRepo.GetVersions(ctx, ruleID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}
	return versions, nil
}

func (s *service) GetAuditLogs(ctx context.Context, ruleID *string, ruleType string, limit int) ([]AuditLog, error) {
	if s.versioningRepo == nil {
		return nil, pkgerrors.ErrInternal.WithDetail("message", "audit logging not enabled")
	}
	if limit <= 0 || limit > constants.MaxLimit {
		limit = constants.DefaultLimit
	}
	logs, err := s.versioningRepo.GetAuditLogs(ctx, ruleID, ruleType, limit)
	if err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}
	return logs, nil
}

func (s *service) CreateEnrichmentRule(ctx context.Context, req CreateEnrichmentRuleRequest) (*EnrichmentRule, error) {
	if err := ValidateEnrichmentRule(req); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrValidation)
	}

	if s.enrichmentRepo == nil {
		return nil, pkgerrors.ErrInternal.WithDetail("message", "enrichment repository not configured")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &EnrichmentRule{
		Name:            req.Name,
		FieldToEnrich:   req.FieldToEnrich,
		SourceType:      req.SourceType,
		SourceConfig:    req.SourceConfig,
		Transformations: req.Transformations,
		CacheTTLSeconds: req.CacheTTLSeconds,
		ErrorHandling:   req.ErrorHandling,
		FallbackValue:   req.FallbackValue,
		Priority:        req.Priority,
		Enabled:         enabled,
	}

	if err := s.enrichmentRepo.CreateEnrichmentRule(ctx, rule); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}

	if s.configEventProducer != nil {
		_ = s.configEventProducer.PublishEnrichmentRuleEvent(ctx, models.ActionCreate, rule.ID, getChangedBy(ctx))
	}

	return rule, nil
}

func (s *service) ListEnrichmentRules(ctx context.Context) ([]EnrichmentRule, error) {
	if s.enrichmentRepo == nil {
		return nil, pkgerrors.ErrInternal.WithDetail("message", "enrichment repository not configured")
	}

	rules, err := s.enrichmentRepo.ListEnrichmentRules(ctx)
	if err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}
	return rules, nil
}

func (s *service) GetEnrichmentRule(ctx context.Context, id string) (*EnrichmentRule, error) {
	if s.enrichmentRepo == nil {
		return nil, pkgerrors.ErrInternal.WithDetail("message", "enrichment repository not configured")
	}

	rule, err := s.enrichmentRepo.GetEnrichmentRule(ctx, id)
	if err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}
	if rule == nil {
		return nil, pkgerrors.ErrNotFound.WithDetail("id", id)
	}
	return rule, nil
}

func (s *service) UpdateEnrichmentRule(ctx context.Context, id string, req UpdateEnrichmentRuleRequest) (*EnrichmentRule, error) {
	if err := ValidateUpdateEnrichmentRule(req); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrValidation)
	}

	if s.enrichmentRepo == nil {
		return nil, pkgerrors.ErrInternal.WithDetail("message", "enrichment repository not configured")
	}

	oldRule, err := s.enrichmentRepo.GetEnrichmentRule(ctx, id)
	if err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}
	if oldRule == nil {
		return nil, pkgerrors.ErrNotFound.WithDetail("id", id)
	}

	if req.Name != nil {
		oldRule.Name = *req.Name
	}
	if req.FieldToEnrich != nil {
		oldRule.FieldToEnrich = *req.FieldToEnrich
	}
	if req.SourceType != nil {
		oldRule.SourceType = *req.SourceType
	}
	if req.SourceConfig != nil {
		oldRule.SourceConfig = *req.SourceConfig
	}
	if req.Transformations != nil {
		oldRule.Transformations = *req.Transformations
	}
	if req.CacheTTLSeconds != nil {
		oldRule.CacheTTLSeconds = *req.CacheTTLSeconds
	}
	if req.ErrorHandling != nil {
		oldRule.ErrorHandling = *req.ErrorHandling
	}
	if req.FallbackValue != nil {
		oldRule.FallbackValue = *req.FallbackValue
	}
	if req.Priority != nil {
		oldRule.Priority = *req.Priority
	}
	if req.Enabled != nil {
		oldRule.Enabled = *req.Enabled
	}

	if err := s.enrichmentRepo.UpdateEnrichmentRule(ctx, oldRule); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}

	if s.configEventProducer != nil {
		_ = s.configEventProducer.PublishEnrichmentRuleEvent(ctx, models.ActionUpdate, oldRule.ID, getChangedBy(ctx))
	}

	return oldRule, nil
}

func (s *service) DeleteEnrichmentRule(ctx context.Context, id string) error {
	if s.enrichmentRepo == nil {
		return pkgerrors.ErrInternal.WithDetail("message", "enrichment repository not configured")
	}

	rule, err := s.enrichmentRepo.GetEnrichmentRule(ctx, id)
	if err != nil {
		return pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}
	if rule == nil {
		return pkgerrors.ErrNotFound.WithDetail("id", id)
	}

	if err := s.enrichmentRepo.DeleteEnrichmentRule(ctx, id); err != nil {
		return pkgerrors.Wrap(err, pkgerrors.ErrInternal)
	}

	if s.configEventProducer != nil {
		_ = s.configEventProducer.PublishEnrichmentRuleEvent(ctx, models.ActionDelete, id, getChangedBy(ctx))
	}

	return nil
}

func (s *service) GetDeduplicationConfig(ctx context.Context) (*DeduplicationConfig, error) {
	s.dedupConfigMu.RLock()
	defer s.dedupConfigMu.RUnlock()

	if s.dedupConfig == nil {
		return nil, pkgerrors.ErrNotFound.WithDetail("message", "deduplication config not initialized")
	}

	config := &DeduplicationConfig{
		HashAlgorithm: s.dedupConfig.HashAlgorithm,
		TTLSeconds:    s.dedupConfig.TTLSeconds,
		OnRedisError:  s.dedupConfig.OnRedisError,
		FieldsToHash:  make([]string, len(s.dedupConfig.FieldsToHash)),
	}
	copy(config.FieldsToHash, s.dedupConfig.FieldsToHash)

	return config, nil
}

func (s *service) UpdateDeduplicationConfig(ctx context.Context, req UpdateDeduplicationConfigRequest) (*DeduplicationConfig, error) {
	if err := ValidateDeduplicationConfig(req); err != nil {
		return nil, pkgerrors.Wrap(err, pkgerrors.ErrValidation)
	}

	s.dedupConfigMu.Lock()
	defer s.dedupConfigMu.Unlock()

	if s.dedupConfig == nil {
		s.dedupConfig = &DeduplicationConfig{
			HashAlgorithm: "md5",
			TTLSeconds:    constants.DefaultTTLSeconds,
			OnRedisError:  "allow",
			FieldsToHash:  []string{"id", "source"},
		}
	}

	// Update fields
	if req.HashAlgorithm != nil {
		s.dedupConfig.HashAlgorithm = *req.HashAlgorithm
	}
	if req.TTLSeconds != nil {
		s.dedupConfig.TTLSeconds = *req.TTLSeconds
	}
	if req.OnRedisError != nil {
		s.dedupConfig.OnRedisError = *req.OnRedisError
	}
	if req.FieldsToHash != nil {
		s.dedupConfig.FieldsToHash = *req.FieldsToHash
	}

	if s.configEventProducer != nil {
		eventMetadata := map[string]interface{}{
			"fields_to_hash": s.dedupConfig.FieldsToHash,
			"hash_algorithm": s.dedupConfig.HashAlgorithm,
			"ttl_seconds":    s.dedupConfig.TTLSeconds,
		}

		_ = s.configEventProducer.PublishDedupConfigEvent(ctx, models.ActionUpdate, getChangedBy(ctx), eventMetadata)
	}

	config := &DeduplicationConfig{
		HashAlgorithm: s.dedupConfig.HashAlgorithm,
		TTLSeconds:    s.dedupConfig.TTLSeconds,
		OnRedisError:  s.dedupConfig.OnRedisError,
		FieldsToHash:  make([]string, len(s.dedupConfig.FieldsToHash)),
	}
	copy(config.FieldsToHash, s.dedupConfig.FieldsToHash)

	return config, nil
}

func (s *service) handleNotFoundError(err error, id string) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "not found") {
		return pkgerrors.ErrNotFound.WithDetail("id", id)
	}
	return pkgerrors.Wrap(err, pkgerrors.ErrInternal)
}

func (s *service) createVersionAndAudit(ctx context.Context, rule *FilteringRule, action string, oldValue map[string]interface{}) {
	if !s.auditEnabled || s.versioningRepo == nil {
		return
	}

	ruleJSON, err := ruleToJSON(rule)
	if err != nil {
		return
	}

	version := s.buildVersion(ctx, rule, ruleJSON)
	if err := s.versioningRepo.CreateVersion(ctx, version); err != nil {
		return
	}

	newValue, err := s.ruleToMap(rule)
	if err != nil {
		return
	}

	auditLog := s.buildAuditLog(rule.ID, "filtering", action, oldValue, newValue, getChangedBy(ctx))
	_ = s.versioningRepo.CreateAuditLog(ctx, auditLog)
}

func (s *service) buildVersion(ctx context.Context, rule *FilteringRule, ruleJSON string) *RuleVersion {
	version := 1
	if s.versioningRepo != nil {
		if nextVersion, err := s.versioningRepo.GetNextVersion(ctx, rule.ID); err == nil {
			version = nextVersion
		}
	}

	return &RuleVersion{
		RuleID:    rule.ID,
		RuleType:  "filtering",
		RuleData:  ruleJSON,
		Version:   version,
		ChangedBy: getChangedBy(ctx),
	}
}

func (s *service) buildAuditLog(ruleID, ruleType, action string, oldValue, newValue map[string]interface{}, changedBy string) *AuditLog {
	return &AuditLog{
		RuleID:    &ruleID,
		RuleType:  ruleType,
		Action:    action,
		OldValue:  oldValue,
		NewValue:  newValue,
		ChangedBy: changedBy,
	}
}

func (s *service) ruleToMap(rule *FilteringRule) (map[string]interface{}, error) {
	ruleData, err := json.Marshal(rule)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(ruleData, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *service) publishConfigEvent(ctx context.Context, action, ruleID string) {
	if s.configEventProducer != nil {
		_ = s.configEventProducer.PublishFilteringRuleEvent(ctx, action, ruleID, getChangedBy(ctx))
	}
}

func (s *service) updateFilteringRuleFields(rule *FilteringRule, req UpdateFilteringRuleRequest) {
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Expression != nil {
		rule.Expression = *req.Expression
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
}

func (s *service) copyFilteringRule(rule *FilteringRule) *FilteringRule {
	return &FilteringRule{
		ID:         rule.ID,
		Name:       rule.Name,
		Expression: rule.Expression,
		Priority:   rule.Priority,
		Enabled:    rule.Enabled,
		CreatedAt:  rule.CreatedAt,
		UpdatedAt:  rule.UpdatedAt,
	}
}

func getEnabledValue(reqEnabled *bool) bool {
	if reqEnabled == nil {
		return true
	}
	return *reqEnabled
}

func getChangedBy(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return "system"
}
