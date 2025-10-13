package deduplication

import (
	"context"
	"fmt"
	"sync"
	"time"

	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/logger"
	"yeti/pkg/metrics"
	"yeti/pkg/models"
	"yeti/pkg/tracing"
)

type redisErrorHandlingStatus int

const (
	redisErrorHandlingDeny redisErrorHandlingStatus = iota
	redisErrorHandlingAllow
)

// Service implements the deduplication service
type Service struct {
	repo             Repository
	hasher           *Hasher
	cfg              config.DeduplicationConfig
	fieldsToHash     []string
	logger           logger.Logger
	fieldsMu         sync.RWMutex       // Protect fieldsToHash from concurrent access
	stopCacheMetrics chan struct{}      // Channel to stop cache metrics updater
	cancelMetricsCtx context.CancelFunc // Cancel function for metrics context
}

// NewService creates a new deduplication service instance
func NewService(repo Repository, cfg config.DeduplicationConfig, log logger.Logger) *Service {
	// Use configured fields or default to ["id", "source"]
	fieldsToHash := cfg.FieldsToHash
	if len(fieldsToHash) == 0 {
		fieldsToHash = []string{"id", "source"}
		log.Infow("No fields_to_hash configured, using defaults", "fields", fieldsToHash)
	}

	// Create a cancellable context for the metrics updater goroutine
	ctx, cancel := context.WithCancel(context.Background())

	s := &Service{
		repo:             repo,
		hasher:           NewHasher(cfg.HashAlgorithm),
		cfg:              cfg,
		fieldsToHash:     fieldsToHash,
		logger:           log,
		stopCacheMetrics: make(chan struct{}),
		cancelMetricsCtx: cancel,
	}

	// Start background goroutine to update cache size metric
	go s.updateCacheSizeMetrics(ctx)

	return s
}

// Process checks if the message is unique.
func (s *Service) Process(ctx context.Context, msg models.MessageEnvelope) (bool, error) {
	ctx, span := tracing.GetTracer("dedup-service").Start(ctx, "deduplication.process")
	defer span.End()

	if err := ctx.Err(); err != nil {
		return false, err
	}

	messageData := s.buildMessageData(msg)
	fieldsToHash := s.getFieldsToHash()

	hash, err := s.computeHash(messageData, fieldsToHash, msg.ID)
	if err != nil {
		return false, err
	}

	if err := ctx.Err(); err != nil {
		return false, err
	}

	key := constants.CacheKeyPrefixDedup + hash
	start := time.Now()
	success, err := s.repo.SetNX(ctx, key, time.Now().Unix(), time.Duration(s.cfg.TTLSeconds)*time.Second)
	duration := time.Since(start)

	if err != nil {
		return s.handleRedisError(ctx, err, duration, msg.ID)
	}

	s.recordMetrics(duration, success)
	return success, nil
}

func (s *Service) buildMessageData(msg models.MessageEnvelope) map[string]interface{} {
	messageData := make(map[string]interface{}, len(msg.Payload)+2)
	messageData["id"] = msg.ID
	messageData["source"] = msg.Source
	for key, value := range msg.Payload {
		messageData[key] = value
	}
	return messageData
}

func (s *Service) getFieldsToHash() []string {
	s.fieldsMu.RLock()
	defer s.fieldsMu.RUnlock()

	fields := make([]string, len(s.fieldsToHash))
	copy(fields, s.fieldsToHash)
	return fields
}

func (s *Service) computeHash(messageData map[string]interface{}, fieldsToHash []string, msgID string) (string, error) {
	hash, err := s.hasher.ComputeHash(messageData, fieldsToHash)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash for message %s: %w", msgID, err)
	}
	return hash, nil
}

func (s *Service) handleRedisError(ctx context.Context, err error, duration time.Duration, msgID string) (bool, error) {
	s.recordMetricsWithStatus(duration, "error")
	status := s.getRedisErrorHandlingStatus(ctx, err, msgID)

	if status == redisErrorHandlingAllow {
		return true, nil
	}
	return false, fmt.Errorf("redis error during dedup check for message %s: %w", msgID, err)
}

func (s *Service) getRedisErrorHandlingStatus(ctx context.Context, err error, msgID string) redisErrorHandlingStatus {
	if s.cfg.OnRedisError == constants.FallbackAllow {
		metrics.FallbackUsageTotal.WithLabelValues("deduplication", "allow_on_error", err.Error()).Inc()
		s.logger.WarnwCtx(ctx, "Redis error during dedup check, allowing message (fallback: allow)",
			"error", err,
		)
		return redisErrorHandlingAllow
	}

	metrics.FallbackUsageTotal.WithLabelValues("deduplication", "deny_on_error", err.Error()).Inc()
	return redisErrorHandlingDeny
}

func (s *Service) recordMetrics(duration time.Duration, isUnique bool) {
	status := "duplicate"
	if isUnique {
		status = "unique"
	}
	s.recordMetricsWithStatus(duration, status)
}

func (s *Service) recordMetricsWithStatus(duration time.Duration, status string) {
	metrics.DeduplicateMessagesTotal.WithLabelValues(status).Inc()
	metrics.ObserveDedupDuration(duration, status)
}

// UpdateFieldsToHash updates the list of fields used for hashing
func (s *Service) UpdateFieldsToHash(fields []string) error {
	if len(fields) == 0 {
		return fmt.Errorf("fields list cannot be empty")
	}

	// Create a copy to prevent external modification
	fieldsCopy := make([]string, len(fields))
	copy(fieldsCopy, fields)

	s.fieldsMu.Lock()
	s.fieldsToHash = fieldsCopy
	s.fieldsMu.Unlock()

	// Note: UpdateFieldsToHash doesn't have context, use background context
	s.logger.Infow("Updated fields to hash", "fields", fieldsCopy)
	return nil
}

// GetFieldsToHash returns the current list of fields used for hashing
func (s *Service) GetFieldsToHash() []string {
	s.fieldsMu.RLock()
	defer s.fieldsMu.RUnlock()

	// Return a copy to prevent external modification
	fields := make([]string, len(s.fieldsToHash))
	copy(fields, s.fieldsToHash)
	return fields
}

// updateCacheSizeMetrics periodically updates the DedupCacheSize metric
func (s *Service) updateCacheSizeMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}
			size, err := s.repo.GetCacheSize(ctx, constants.CacheKeyPrefixDedup)
			if err != nil {
				// Check context cancellation after error to avoid infinite loop
				if ctx.Err() != nil {
					return
				}
				s.logger.Debugw("Failed to get cache size for metrics",
					"error", err,
				)
				continue
			}
			// Check context cancellation after successful call
			if ctx.Err() != nil {
				return
			}
			metrics.SetDedupCacheSize(size)
		case <-s.stopCacheMetrics:
			return
		case <-ctx.Done():
			return
		}
	}
}

// StopCacheMetricsUpdater stops the background cache metrics updater
func (s *Service) StopCacheMetricsUpdater() {
	if s.cancelMetricsCtx != nil {
		s.cancelMetricsCtx()
	}
	close(s.stopCacheMetrics)
}
