package retry

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type RetryableError interface {
	error
	IsRetryable() bool
}

type retryableError struct {
	err error
}

func (e *retryableError) Error() string {
	return e.err.Error()
}

func (e *retryableError) IsRetryable() bool {
	return true
}

func (e *retryableError) Unwrap() error {
	return e.err
}

func NewRetryableError(err error) RetryableError {
	if err == nil {
		return nil
	}
	return &retryableError{err: err}
}

type FatalError interface {
	error
	IsFatal() bool
}

type fatalError struct {
	err error
}

func (e *fatalError) Error() string {
	return e.err.Error()
}

func (e *fatalError) IsFatal() bool {
	return true
}

func (e *fatalError) Unwrap() error {
	return e.err
}

func NewFatalError(err error) FatalError {
	if err == nil {
		return nil
	}
	return &fatalError{err: err}
}

type Policy struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxElapsedTime  time.Duration
}

func DefaultPolicy() Policy {
	return Policy{
		MaxAttempts:     3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
		MaxElapsedTime:  5 * time.Minute,
	}
}

func Retry(ctx context.Context, policy Policy, fn func() error) error {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 3
	}

	var b backoff.BackOff
	if policy.MaxElapsedTime > 0 {
		b = ExponentialBackoffWithMaxElapsed(
			policy.InitialInterval,
			policy.MaxInterval,
			policy.MaxElapsedTime,
			policy.Multiplier,
		)
	} else {
		b = ExponentialBackoff(
			policy.InitialInterval,
			policy.MaxInterval,
			policy.Multiplier,
		)
	}

	b = backoff.WithContext(b, ctx)
	b = backoff.WithMaxRetries(b, uint64(policy.MaxAttempts-1))

	attempt := 0
	operation := func() error {
		attempt++
		err := fn()

		if err == nil {
			return nil
		}

		var fatalErr FatalError
		if errors.As(err, &fatalErr) {
			return backoff.Permanent(err)
		}

		var retryableErr RetryableError
		if !errors.As(err, &retryableErr) {
			// Default: treat as retryable
			return NewRetryableError(err)
		}

		return err
	}

	return backoff.Retry(operation, b)
}

func RetryWithCallback(ctx context.Context, policy Policy, fn func() error, onRetry func(attempt int, err error, nextDelay time.Duration)) error {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = 3
	}

	var b backoff.BackOff
	if policy.MaxElapsedTime > 0 {
		b = ExponentialBackoffWithMaxElapsed(
			policy.InitialInterval,
			policy.MaxInterval,
			policy.MaxElapsedTime,
			policy.Multiplier,
		)
	} else {
		b = ExponentialBackoff(
			policy.InitialInterval,
			policy.MaxInterval,
			policy.Multiplier,
		)
	}

	b = backoff.WithContext(b, ctx)
	b = backoff.WithMaxRetries(b, uint64(policy.MaxAttempts-1))

	attempt := 0
	operation := func() error {
		attempt++
		err := fn()

		if err == nil {
			return nil
		}

		var fatalErr FatalError
		if errors.As(err, &fatalErr) {
			return backoff.Permanent(err)
		}

		var retryableErr RetryableError
		if !errors.As(err, &retryableErr) {
			// Default: treat as retryable
			err = NewRetryableError(err)
		}

		if onRetry != nil && attempt < policy.MaxAttempts {
			nextDelay := CalculateBackoffDuration(attempt, policy.InitialInterval, policy.Multiplier, policy.MaxInterval)
			onRetry(attempt, err, nextDelay)
		}

		return err
	}

	return backoff.Retry(operation, b)
}
