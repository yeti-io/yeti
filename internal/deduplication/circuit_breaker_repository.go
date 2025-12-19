package deduplication

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"yeti/internal/config"
	"yeti/pkg/circuitbreaker"
)

type CircuitBreakerRepository struct {
	repo Repository
	cb   *circuitbreaker.Wrapper
}

func NewCircuitBreakerRepository(repo Repository, cfg config.CircuitBreakerConfig) *CircuitBreakerRepository {
	if !cfg.Enabled {
		return &CircuitBreakerRepository{
			repo: repo,
			cb:   nil,
		}
	}

	cbConfig := circuitbreaker.DefaultConfig("redis-dedup")
	if cfg.MaxRequests > 0 {
		cbConfig.MaxRequests = cfg.MaxRequests
	}
	if cfg.Interval > 0 {
		cbConfig.Interval = cfg.Interval
	}
	if cfg.Timeout > 0 {
		cbConfig.Timeout = cfg.Timeout
	}
	if cfg.FailureRatio > 0 && cfg.MinRequests > 0 {
		cbConfig.ReadyToTrip = func(counts gobreaker.Counts) bool {
			if counts.Requests < uint32(cfg.MinRequests) {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= cfg.FailureRatio
		}
	}

	return &CircuitBreakerRepository{
		repo: repo,
		cb:   circuitbreaker.NewWrapper(cbConfig),
	}
}

func (r *CircuitBreakerRepository) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if r.cb == nil {
		return r.repo.SetNX(ctx, key, value, ttl)
	}

	result, err := r.cb.ExecuteWithContext(ctx, func() (interface{}, error) {
		return r.repo.SetNX(ctx, key, value, ttl)
	})

	r.cb.RecordRequest(err == nil)

	if err != nil {
		if r.cb.IsOpen() {
			return false, fmt.Errorf("circuit breaker is open for redis-dedup: %w", err)
		}
		return false, err
	}

	success, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("repository returned invalid result type")
	}

	return success, nil
}

func (r *CircuitBreakerRepository) State() string {
	if r.cb == nil {
		return "disabled"
	}
	return r.cb.State().String()
}

func (r *CircuitBreakerRepository) IsOpen() bool {
	if r.cb == nil {
		return false
	}
	return r.cb.IsOpen()
}

func (r *CircuitBreakerRepository) GetCacheSize(ctx context.Context, prefix string) (int, error) {
	if r.cb == nil {
		return r.repo.GetCacheSize(ctx, prefix)
	}

	result, err := r.cb.ExecuteWithContext(ctx, func() (interface{}, error) {
		return r.repo.GetCacheSize(ctx, prefix)
	})

	r.cb.RecordRequest(err == nil)

	if err != nil {
		if r.cb.IsOpen() {
			return 0, fmt.Errorf("circuit breaker is open for redis-dedup: %w", err)
		}
		return 0, err
	}

	size, ok := result.(int)
	if !ok {
		return 0, fmt.Errorf("repository returned invalid result type")
	}

	return size, nil
}
