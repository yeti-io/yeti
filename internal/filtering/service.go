package filtering

import (
	"context"
	"fmt"
	"sync"
	"time"

	"yeti/internal/config"
	"yeti/internal/constants"
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

func getRuleIDs(rules []Rule) []string {
	ids := make([]string, 0, len(rules))
	for _, r := range rules {
		ids = append(ids, r.ID)
	}
	return ids
}

func getRuleNames(rules []Rule) []string {
	names := make([]string, 0, len(rules))
	for _, r := range rules {
		names = append(names, r.Name)
	}
	return names
}

type errorHandlingStatus int

const (
	errorHandlingDeny errorHandlingStatus = iota
	errorHandlingSkip
)

type Service struct {
	repo            Repository
	rules           []Rule
	rulesMu         sync.RWMutex
	filteringConfig config.FilteringConfig
	evaluator       *cel.Evaluator
	logger          logger.Logger
}

func NewService(repo Repository, cfg config.FilteringConfig, log logger.Logger) (*Service, error) {
	evaluator, err := cel.NewEvaluator()
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL evaluator: %w", err)
	}

	return &Service{
		repo:            repo,
		filteringConfig: cfg,
		rules:           make([]Rule, 0),
		evaluator:       evaluator,
		logger:          log,
	}, nil
}

func (s *Service) Filter(ctx context.Context, msg models.MessageEnvelope) (bool, []string, error) {
	ctx, span := tracing.GetTracer("filtering-service").Start(ctx, "filtering.filter")
	defer span.End()

	s.logger.DebugwCtx(ctx, "Filtering message",
		"message_id", msg.ID,
		"source", msg.Source,
		"payload_keys", getPayloadKeys(msg.Payload),
	)

	rules := s.getActiveRules()
	s.logger.DebugwCtx(ctx, "Active filtering rules loaded",
		"rules_count", len(rules),
		"rule_ids", getRuleIDs(rules),
	)

	appliedRules := make([]string, 0, len(rules))
	start := time.Now()

	passed, appliedRules, err := s.evaluateRules(ctx, rules, msg, &appliedRules)

	duration := time.Since(start)
	s.recordMetrics(duration, passed)
	
	s.logger.DebugwCtx(ctx, "Filtering completed",
		"message_id", msg.ID,
		"passed", passed,
		"applied_rules_count", len(appliedRules),
		"applied_rule_ids", appliedRules,
		"duration_ms", duration.Milliseconds(),
	)

	return passed, appliedRules, err
}

func (s *Service) getActiveRules() []Rule {
	s.rulesMu.RLock()
	defer s.rulesMu.RUnlock()

	rules := make([]Rule, len(s.rules))
	copy(rules, s.rules)
	return rules
}

func (s *Service) evaluateRules(ctx context.Context, rules []Rule, msg models.MessageEnvelope, appliedRules *[]string) (bool, []string, error) {
	ctx, span := tracing.GetTracer("filtering-service").Start(ctx, "filtering.evaluate_rules")
	defer span.End()

	for i, rule := range rules {
		if err := ctx.Err(); err != nil {
			return false, nil, err
		}

		s.logger.DebugwCtx(ctx, "Evaluating filtering rule",
			"rule_index", i+1,
			"total_rules", len(rules),
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"expression", rule.Expression,
			"priority", rule.Priority,
			"enabled", rule.Enabled,
		)

		result, err := s.evaluator.EvaluateFilter(ctx, rule.Expression, msg)
		if err != nil {
			status := s.handleEvaluationError(ctx, rule, err)
			if status == errorHandlingDeny {
				s.logger.DebugwCtx(ctx, "Message denied due to evaluation error",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
				)
				return false, *appliedRules, nil
			}
			continue
		}

		s.logger.DebugwCtx(ctx, "Rule evaluation result",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"result", result,
		)

		if !result {
			s.logger.DebugwCtx(ctx, "Rule filtered message",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
			)
			return false, *appliedRules, nil
		}

		*appliedRules = append(*appliedRules, rule.ID)
		s.logger.DebugwCtx(ctx, "Rule passed, message continues",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"total_applied_rules", len(*appliedRules),
		)
	}

	s.logger.DebugwCtx(ctx, "All rules passed",
		"total_rules", len(rules),
		"applied_rules_count", len(*appliedRules),
	)

	return true, *appliedRules, nil
}

func (s *Service) handleEvaluationError(ctx context.Context, rule Rule, err error) errorHandlingStatus {
	s.logger.ErrorwCtx(ctx, "Rule evaluation error",
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"error", err,
	)

	switch s.filteringConfig.Fallback.OnError {
	case constants.FallbackAllow:
		metrics.FallbackUsageTotal.WithLabelValues("filtering", "allow_on_error", "evaluation_error").Inc()
		s.logger.WarnwCtx(ctx, "Evaluation error, allowing message (fallback: allow)",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"error", err,
		)
		return errorHandlingSkip
	case constants.FallbackDeny:
		metrics.FallbackUsageTotal.WithLabelValues("filtering", "deny_on_error", "evaluation_error").Inc()
		s.logger.WarnwCtx(ctx, "Evaluation error, denying message (fallback: deny)",
			"rule_id", rule.ID,
			"rule_name", rule.Name,
			"error", err,
		)
		return errorHandlingDeny
	default:
		return errorHandlingSkip
	}
}

func (s *Service) recordMetrics(duration time.Duration, passed bool) {
	status := "passed"
	if !passed {
		status = "filtered"
	}
	metrics.FilteringMessagesTotal.WithLabelValues(status).Inc()
	metrics.ObserveFilteringDuration(duration, status)
}

func (s *Service) ReloadRules(ctx context.Context) error {
	rules, err := s.loadRules(ctx)
	if err != nil {
		return err
	}

	s.updateRules(ctx, rules)
	return nil
}

func (s *Service) loadRules(ctx context.Context) ([]Rule, error) {
	s.logger.DebugwCtx(ctx, "Loading rules from database")
	rules, err := s.repo.GetActiveRules(ctx)
	if err != nil {
		s.logger.ErrorwCtx(ctx, "Failed to load rules from database",
			"error", err,
		)
		return nil, err
	}
	
	s.logger.DebugwCtx(ctx, "Rules loaded from database",
		"rules_count", len(rules),
		"rule_names", getRuleNames(rules),
	)
	
	return rules, nil
}

func (s *Service) updateRules(ctx context.Context, rules []Rule) {
	s.rulesMu.Lock()
	oldCount := len(s.rules)
	s.rules = rules
	s.rulesMu.Unlock()

	metrics.SetFilteringActiveRules(len(rules))
	s.logger.InfowCtx(ctx, "Successfully reloaded rules",
		"old_rules_count", oldCount,
		"new_rules_count", len(rules),
		"rule_ids", getRuleIDs(rules),
	)
}

func (s *Service) StartReloader(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(s.filteringConfig.Reload.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	if err := s.ReloadRules(ctx); err != nil {
		s.logger.ErrorwCtx(ctx, "Failed to reload rules",
			"error", err,
		)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.ReloadRules(ctx); err != nil {
				s.logger.ErrorwCtx(ctx, "Failed to reload rules",
					"error", err,
				)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
