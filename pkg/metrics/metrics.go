package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	FilteringMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "filtering_messages_total",
			Help: "Total number of messages processed by filtering service",
		},
		[]string{"status"},
	)

	DeduplicateMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dedup_messages_total",
			Help: "Total number of messages processed by deduplication service",
		},
		[]string{"status"},
	)

	EnrichmentMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_messages_total",
			Help: "Total number of messages processed by enrichment service",
		},
		[]string{"status"},
	)

	FilteringProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "filtering_processing_duration_ms",
			Help:    "Processing duration for filtering service in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"status"},
	)

	DedupProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dedup_processing_duration_ms",
			Help:    "Processing duration for deduplication service in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"status"},
	)

	EnrichmentProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "enrichment_processing_duration_ms",
			Help:    "Processing duration for enrichment service in milliseconds",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"status"},
	)

	FilteringActiveRules = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "filtering_active_rules",
			Help: "Number of active filtering rules",
		},
	)

	EnrichmentActiveRules = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "enrichment_active_rules",
			Help: "Number of active enrichment rules",
		},
	)

	DedupCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "dedup_cache_size",
			Help: "Approximate size of deduplication cache (number of unique hashes)",
		},
	)

	EnrichmentCacheHitRate = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "enrichment_cache_hit_rate",
			Help: "Cache hit rate for enrichment service (0.0 to 1.0)",
		},
	)

	RetryAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_attempts_total",
			Help: "Total number of retry attempts",
		},
		[]string{"service", "topic"},
	)

	DLQMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dlq_messages_total",
			Help: "Total number of messages sent to DLQ",
		},
		[]string{"service", "topic", "reason"},
	)

	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
		},
		[]string{"name"},
	)

	CircuitBreakerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total number of requests through circuit breaker",
		},
		[]string{"name", "state"},
	)

	CircuitBreakerFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Total number of failures through circuit breaker",
		},
		[]string{"name"},
	)

	RateLimitRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_requests_total",
			Help: "Total number of requests checked against rate limit",
		},
		[]string{"status"}, // status: "allowed" or "limited"
	)

	FallbackUsageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fallback_usage_total",
			Help: "Total number of times fallback strategies were used",
		},
		[]string{"service", "strategy", "reason"}, // service: "enrichment", "deduplication", "filtering"
	)
)

func RegisterFilteringMetrics() {
	_ = prometheus.Register(FilteringMessagesTotal)
	_ = prometheus.Register(FilteringProcessingDuration)
	_ = prometheus.Register(FilteringActiveRules)
	_ = prometheus.Register(FallbackUsageTotal)
}

func RegisterDedupMetrics() {
	_ = prometheus.Register(DeduplicateMessagesTotal)
	_ = prometheus.Register(DedupProcessingDuration)
	_ = prometheus.Register(DedupCacheSize)
	_ = prometheus.Register(FallbackUsageTotal)
}

func RegisterEnrichmentMetrics() {
	_ = prometheus.Register(EnrichmentMessagesTotal)
	_ = prometheus.Register(EnrichmentProcessingDuration)
	_ = prometheus.Register(EnrichmentActiveRules)
	_ = prometheus.Register(EnrichmentCacheHitRate)
	_ = prometheus.Register(FallbackUsageTotal)
}

func RegisterBrokerMetrics() {
	_ = prometheus.Register(RetryAttemptsTotal)
	_ = prometheus.Register(DLQMessagesTotal)
}

func RegisterCircuitBreakerMetrics() {
	_ = prometheus.Register(CircuitBreakerState)
	_ = prometheus.Register(CircuitBreakerRequests)
	_ = prometheus.Register(CircuitBreakerFailures)
}

func RegisterManagementMetrics() {
	_ = prometheus.Register(RateLimitRequestsTotal)
}

func ObserveFilteringDuration(duration time.Duration, status string) {
	FilteringProcessingDuration.WithLabelValues(status).Observe(float64(duration.Milliseconds()))
}

func ObserveDedupDuration(duration time.Duration, status string) {
	DedupProcessingDuration.WithLabelValues(status).Observe(float64(duration.Milliseconds()))
}

func ObserveEnrichmentDuration(duration time.Duration, status string) {
	EnrichmentProcessingDuration.WithLabelValues(status).Observe(float64(duration.Milliseconds()))
}

func SetFilteringActiveRules(count int) {
	FilteringActiveRules.Set(float64(count))
}

func SetEnrichmentActiveRules(count int) {
	EnrichmentActiveRules.Set(float64(count))
}

func SetDedupCacheSize(size int) {
	DedupCacheSize.Set(float64(size))
}

func SetEnrichmentCacheHitRate(rate float64) {
	EnrichmentCacheHitRate.Set(rate)
}
