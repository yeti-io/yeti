package retry

import (
	"math"
	"time"

	"github.com/cenkalti/backoff/v4"
)

func ExponentialBackoff(initialInterval, maxInterval time.Duration, multiplier float64) backoff.BackOff {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = initialInterval
	exp.MaxInterval = maxInterval
	exp.Multiplier = multiplier
	exp.MaxElapsedTime = 0
	return exp
}

func ExponentialBackoffWithMaxElapsed(initialInterval, maxInterval, maxElapsed time.Duration, multiplier float64) backoff.BackOff {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = initialInterval
	exp.MaxInterval = maxInterval
	exp.Multiplier = multiplier
	exp.MaxElapsedTime = maxElapsed
	return exp
}

func CalculateBackoffDuration(attempt int, initialInterval time.Duration, multiplier float64, maxInterval time.Duration) time.Duration {
	duration := float64(initialInterval) * math.Pow(multiplier, float64(attempt))
	if duration > float64(maxInterval) {
		return maxInterval
	}
	return time.Duration(duration)
}
