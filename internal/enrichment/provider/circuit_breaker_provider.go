package provider

import (
	"context"
	"fmt"

	"yeti/pkg/circuitbreaker"
)

type CircuitBreakerProvider struct {
	provider DataProvider
	cb       *circuitbreaker.Wrapper
	name     string
}

func NewCircuitBreakerProvider(provider DataProvider, name string, cfg circuitbreaker.Config) *CircuitBreakerProvider {
	return &CircuitBreakerProvider{
		provider: provider,
		cb:       circuitbreaker.NewWrapper(cfg),
		name:     name,
	}
}

func (p *CircuitBreakerProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (map[string]interface{}, error) {
	result, err := p.cb.ExecuteWithContext(ctx, func() (interface{}, error) {
		return p.provider.Fetch(ctx, config, fieldValue)
	})

	p.cb.RecordRequest(err == nil)

	if err != nil {
		if p.cb.IsOpen() {
			return nil, fmt.Errorf("circuit breaker is open for %s: %w", p.name, err)
		}
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("provider returned nil result")
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("provider returned invalid result type")
	}

	return data, nil
}

func (p *CircuitBreakerProvider) State() string {
	return p.cb.State().String()
}

func (p *CircuitBreakerProvider) IsOpen() bool {
	return p.cb.IsOpen()
}
