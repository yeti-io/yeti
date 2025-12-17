package circuitbreaker

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
	"yeti/pkg/metrics"
)

// Config defines circuit breaker configuration
type Config struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts gobreaker.Counts) bool
	OnStateChange func(name string, from, to gobreaker.State)
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig(name string) Config {
	return Config{
		Name:        name,
		MaxRequests: 3,
		Interval:    60 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.5
		},
	}
}

// Wrapper wraps a function with circuit breaker logic
type Wrapper struct {
	cb *gobreaker.CircuitBreaker
}

// NewWrapper creates a new circuit breaker wrapper
func NewWrapper(cfg Config) *Wrapper {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
	}

	if cfg.ReadyToTrip != nil {
		settings.ReadyToTrip = cfg.ReadyToTrip
	}

	// Always update metrics on state change, even if user provides custom handler
	settings.OnStateChange = func(name string, from, to gobreaker.State) {
		// Update metrics first
		updateCircuitBreakerMetrics(name, to)
		// Then call user's handler if provided
		if cfg.OnStateChange != nil {
			cfg.OnStateChange(name, from, to)
		}
	}

	cb := gobreaker.NewCircuitBreaker(settings)
	
	// Initialize metrics
	updateCircuitBreakerMetrics(cfg.Name, cb.State())

	return &Wrapper{
		cb: cb,
	}
}

// Execute executes a function with circuit breaker protection
func (w *Wrapper) Execute(fn func() (interface{}, error)) (interface{}, error) {
	return w.cb.Execute(fn)
}

// ExecuteWithContext executes a function with circuit breaker protection and context
func (w *Wrapper) ExecuteWithContext(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Execute with circuit breaker
	result, err := w.cb.Execute(func() (interface{}, error) {
		// Check context again inside the function
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return fn()
		}
	})

	return result, err
}

// State returns the current state of the circuit breaker
func (w *Wrapper) State() gobreaker.State {
	return w.cb.State()
}

// Counts returns the current counts of the circuit breaker
func (w *Wrapper) Counts() gobreaker.Counts {
	return w.cb.Counts()
}

// Name returns the name of the circuit breaker
func (w *Wrapper) Name() string {
	return w.cb.Name()
}

// IsOpen returns true if the circuit breaker is in open state
func (w *Wrapper) IsOpen() bool {
	return w.cb.State() == gobreaker.StateOpen
}

// IsHalfOpen returns true if the circuit breaker is in half-open state
func (w *Wrapper) IsHalfOpen() bool {
	return w.cb.State() == gobreaker.StateHalfOpen
}

// IsClosed returns true if the circuit breaker is in closed state
func (w *Wrapper) IsClosed() bool {
	return w.cb.State() == gobreaker.StateClosed
}

// updateCircuitBreakerMetrics updates Prometheus metrics for circuit breaker
func updateCircuitBreakerMetrics(name string, state gobreaker.State) {
	var stateValue float64
	switch state {
	case gobreaker.StateClosed:
		stateValue = 0
	case gobreaker.StateHalfOpen:
		stateValue = 1
	case gobreaker.StateOpen:
		stateValue = 2
	}
	metrics.CircuitBreakerState.WithLabelValues(name).Set(stateValue)
}

// RecordRequest records a request through the circuit breaker
func (w *Wrapper) RecordRequest(success bool) {
	state := w.cb.State().String()
	metrics.CircuitBreakerRequests.WithLabelValues(w.cb.Name(), state).Inc()
	if !success {
		metrics.CircuitBreakerFailures.WithLabelValues(w.cb.Name()).Inc()
	}
}
