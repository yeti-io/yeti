package filtering

import (
	"context"
	"fmt"
	"math/rand"
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

	rules := s.getActiveRules()
	appliedRules := make([]string, 0, len(rules))
	start := time.Now()

	passed, appliedRules, err := s.evaluateRules(ctx, rules, msg, &appliedRules)

	s.recordMetrics(time.Since(start), passed)
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

	for _, rule := range rules {
		if err := ctx.Err(); err != nil {
			return false, nil, err
		}

		result, err := s.evaluator.EvaluateFilter(ctx, rule.Expression, msg)
		if err != nil {
			status := s.handleEvaluationError(ctx, rule, err)
			if status == errorHandlingDeny {
				return false, *appliedRules, nil
			}
			continue
		}

		if !result {
			s.logger.DebugwCtx(ctx, "Rule filtered message",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
			)
			return false, *appliedRules, nil
		}

		*appliedRules = append(*appliedRules, rule.ID)
	}

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

func (s *Service) ReloadRules(ctx context.Context, skipJitter ...bool) error {
	shouldSkipJitter := len(skipJitter) > 0 && skipJitter[0]

	if err := s.applyJitter(ctx, shouldSkipJitter); err != nil {
		return err
	}

	rules, err := s.loadRules(ctx)
	if err != nil {
		return err
	}

	s.updateRules(ctx, rules)
	return nil
}

func (s *Service) applyJitter(ctx context.Context, skipJitter bool) error {
	if skipJitter || s.filteringConfig.Reload.JitterMaxMilliseconds == 0 {
		return nil
	}

	jitter := time.Duration(rand.Intn(s.filteringConfig.Reload.JitterMaxMilliseconds)) * time.Millisecond
	s.logger.DebugwCtx(ctx, "Reload scheduled with jitter",
		"jitter_ms", jitter.Milliseconds(),
	)

	select {
	case <-time.After(jitter):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) loadRules(ctx context.Context) ([]Rule, error) {
	s.logger.DebugwCtx(ctx, "Loading rules from database")
	rules, err := s.repo.GetActiveRules(ctx)
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *Service) updateRules(ctx context.Context, rules []Rule) {
	s.rulesMu.Lock()
	s.rules = rules
	s.rulesMu.Unlock()

	metrics.SetFilteringActiveRules(len(rules))
	s.logger.InfowCtx(ctx, "Successfully reloaded rules",
		"rules_count", len(rules),
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
