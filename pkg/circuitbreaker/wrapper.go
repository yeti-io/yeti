package circuitbreaker

import (
	"context"
	"time"

	"github.com/sony/gobreaker"
	"yeti/pkg/metrics"
)

type Config struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts gobreaker.Counts) bool
	OnStateChange func(name string, from, to gobreaker.State)
}

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

type Wrapper struct {
	cb *gobreaker.CircuitBreaker
}

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

	settings.OnStateChange = func(name string, from, to gobreaker.State) {
		updateCircuitBreakerMetrics(name, to)
		if cfg.OnStateChange != nil {
			cfg.OnStateChange(name, from, to)
		}
	}

	cb := gobreaker.NewCircuitBreaker(settings)

	updateCircuitBreakerMetrics(cfg.Name, cb.State())

	return &Wrapper{
		cb: cb,
	}
}

func (w *Wrapper) Execute(fn func() (interface{}, error)) (interface{}, error) {
	return w.cb.Execute(fn)
}

func (w *Wrapper) ExecuteWithContext(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result, err := w.cb.Execute(func() (interface{}, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return fn()
		}
	})

	return result, err
}

func (w *Wrapper) State() gobreaker.State {
	return w.cb.State()
}

func (w *Wrapper) Counts() gobreaker.Counts {
	return w.cb.Counts()
}

func (w *Wrapper) Name() string {
	return w.cb.Name()
}

func (w *Wrapper) IsOpen() bool {
	return w.cb.State() == gobreaker.StateOpen
}

func (w *Wrapper) IsHalfOpen() bool {
	return w.cb.State() == gobreaker.StateHalfOpen
}

func (w *Wrapper) IsClosed() bool {
	return w.cb.State() == gobreaker.StateClosed
}

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

func (w *Wrapper) RecordRequest(success bool) {
	state := w.cb.State().String()
	metrics.CircuitBreakerRequests.WithLabelValues(w.cb.Name(), state).Inc()
	if !success {
		metrics.CircuitBreakerFailures.WithLabelValues(w.cb.Name()).Inc()
	}
}
