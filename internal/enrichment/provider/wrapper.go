package provider

import (
	"github.com/sony/gobreaker"
	"yeti/internal/config"
	"yeti/pkg/circuitbreaker"
)

func WrapWithCircuitBreaker(p DataProvider, name string, cfg config.CircuitBreakerConfig) DataProvider {
	if !cfg.Enabled {
		return p
	}

	cbConfig := circuitbreaker.DefaultConfig(name)
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

	return NewCircuitBreakerProvider(p, name, cbConfig)
}
