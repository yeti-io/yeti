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

type redisErrorHandlingStatus int

const (
	redisErrorHandlingDeny redisErrorHandlingStatus = iota
	redisErrorHandlingAllow
)

type Service struct {
	repo             Repository
	hasher           *Hasher
	cfg              config.DeduplicationConfig
	fieldsToHash     []string
	logger           logger.Logger
	fieldsMu         sync.RWMutex
	stopCacheMetrics chan struct{}
	cancelMetricsCtx context.CancelFunc
}

func NewService(repo Repository, cfg config.DeduplicationConfig, log logger.Logger) *Service {
	fieldsToHash := cfg.FieldsToHash
	if len(fieldsToHash) == 0 {
		fieldsToHash = []string{"id", "source"}
		log.Infow("No fields_to_hash configured, using defaults", "fields", fieldsToHash)
	}

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

	go s.updateCacheSizeMetrics(ctx)

	return s
}

func (s *Service) Process(ctx context.Context, msg models.MessageEnvelope) (bool, error) {
	ctx, span := tracing.GetTracer("dedup-service").Start(ctx, "deduplication.process")
	defer span.End()

	s.logger.DebugwCtx(ctx, "Processing message for deduplication",
		"message_id", msg.ID,
		"source", msg.Source,
		"payload_keys", getPayloadKeys(msg.Payload),
	)

	if err := ctx.Err(); err != nil {
		return false, err
	}

	messageData := s.buildMessageData(msg)
	fieldsToHash := s.getFieldsToHash()

	s.logger.DebugwCtx(ctx, "Computing hash for message",
		"message_id", msg.ID,
		"fields_to_hash", fieldsToHash,
		"hash_algorithm", s.cfg.HashAlgorithm,
	)

	hash, err := s.computeHash(messageData, fieldsToHash, msg.ID)
	if err != nil {
		s.logger.ErrorwCtx(ctx, "Failed to compute hash",
			"message_id", msg.ID,
			"error", err,
		)
		return false, err
	}

	s.logger.DebugwCtx(ctx, "Hash computed",
		"message_id", msg.ID,
		"hash", hash,
	)

	if err := ctx.Err(); err != nil {
		return false, err
	}

	key := constants.CacheKeyPrefixDedup + hash
	s.logger.DebugwCtx(ctx, "Checking Redis for duplicate",
		"message_id", msg.ID,
		"redis_key", key,
		"ttl_seconds", s.cfg.TTLSeconds,
	)

	start := time.Now()
	success, err := s.repo.SetNX(ctx, key, time.Now().Unix(), time.Duration(s.cfg.TTLSeconds)*time.Second)
	duration := time.Since(start)

	if err != nil {
		s.logger.DebugwCtx(ctx, "Redis SetNX error",
			"message_id", msg.ID,
			"redis_key", key,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return s.handleRedisError(ctx, err, duration, msg.ID)
	}

	s.logger.DebugwCtx(ctx, "Deduplication check completed",
		"message_id", msg.ID,
		"is_unique", success,
		"redis_key", key,
		"duration_ms", duration.Milliseconds(),
	)

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
	s.logger.DebugwCtx(context.Background(), "Built message data for hashing",
		"message_id", msg.ID,
		"data_keys", getMapKeys(messageData),
	)
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

func (s *Service) UpdateFieldsToHash(fields []string) error {
	if len(fields) == 0 {
		return fmt.Errorf("fields list cannot be empty")
	}

	fieldsCopy := make([]string, len(fields))
	copy(fieldsCopy, fields)

	s.fieldsMu.Lock()
	oldFields := make([]string, len(s.fieldsToHash))
	copy(oldFields, s.fieldsToHash)
	s.fieldsToHash = fieldsCopy
	s.fieldsMu.Unlock()

	s.logger.Infow("Updated fields to hash",
		"old_fields", oldFields,
		"new_fields", fieldsCopy,
	)
	s.logger.Debugw("Fields to hash updated",
		"old_fields", oldFields,
		"new_fields", fieldsCopy,
		"fields_count", len(fieldsCopy),
	)
	return nil
}

func (s *Service) GetFieldsToHash() []string {
	s.fieldsMu.RLock()
	defer s.fieldsMu.RUnlock()

	fields := make([]string, len(s.fieldsToHash))
	copy(fields, s.fieldsToHash)
	return fields
}

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
				if ctx.Err() != nil {
					return
				}
				s.logger.Debugw("Failed to get cache size for metrics",
					"error", err,
				)
				continue
			}
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

func (s *Service) StopCacheMetricsUpdater() {
	if s.cancelMetricsCtx != nil {
		s.cancelMetricsCtx()
	}
	close(s.stopCacheMetrics)
}
