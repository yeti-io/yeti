package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	FilteringMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "filtering_messages_total",
			Help: "Total number of messages processed by filtering service (count)",
		},
		[]string{"status"},
	)

	DeduplicateMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dedup_messages_total",
			Help: "Total number of messages processed by deduplication service (count)",
		},
		[]string{"status"},
	)

	EnrichmentMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_messages_total",
			Help: "Total number of messages processed by enrichment service (count)",
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
			Help: "Number of active filtering rules (count)",
		},
	)

	EnrichmentActiveRules = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "enrichment_active_rules",
			Help: "Number of active enrichment rules (count)",
		},
	)

	DedupCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "dedup_cache_size",
			Help: "Approximate size of deduplication cache (count)",
		},
	)

	EnrichmentCacheHitRate = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "enrichment_cache_hit_rate",
			Help: "Cache hit rate for enrichment service (ratio, 0.0 to 1.0)",
		},
	)

	RetryAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "retry_attempts_total",
			Help: "Total number of retry attempts (count)",
		},
		[]string{"service", "topic"},
	)

	DLQMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dlq_messages_total",
			Help: "Total number of messages sent to DLQ (count)",
		},
		[]string{"service", "topic", "reason"},
	)

	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=half-open, 2=open) (state code)",
		},
		[]string{"name"},
	)

	CircuitBreakerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total number of requests through circuit breaker (count)",
		},
		[]string{"name", "state"},
	)

	CircuitBreakerFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Total number of failures through circuit breaker (count)",
		},
		[]string{"name"},
	)

	RateLimitRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_requests_total",
			Help: "Total number of requests checked against rate limit (count)",
		},
		[]string{"status"},
	)

	FallbackUsageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fallback_usage_total",
			Help: "Total number of times fallback strategies were used (count)",
		},
		[]string{"service", "strategy", "reason"},
	)

	KafkaMessagesReadTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_read_total",
			Help: "Total number of messages read from Kafka (count)",
		},
		[]string{"service", "topic"},
	)

	KafkaMessagesWrittenTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_written_total",
			Help: "Total number of messages written to Kafka (count)",
		},
		[]string{"service", "topic"},
	)

	KafkaMessageSizeBytes = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_message_size_bytes",
			Help:    "Size of Kafka messages in bytes",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000},
		},
		[]string{"service", "topic", "direction"},
	)

	KafkaConsumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Kafka consumer lag (difference between latest offset and committed offset) (count)",
		},
		[]string{"service", "topic", "partition"},
	)

	KafkaReadDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_read_duration_ms",
			Help:    "Duration of reading messages from Kafka in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"service", "topic"},
	)

	KafkaWriteDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_write_duration_ms",
			Help:    "Duration of writing messages to Kafka in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"service", "topic"},
	)

	FilteringRuleEvaluationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "filtering_rule_evaluations_total",
			Help: "Total number of filtering rule evaluations (count)",
		},
		[]string{"rule_id", "rule_name", "result"},
	)

	EnrichmentRuleApplicationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_rule_applications_total",
			Help: "Total number of enrichment rule applications (count)",
		},
		[]string{"rule_id", "rule_name", "status"},
	)

	EnrichmentTransformationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_transformations_total",
			Help: "Total number of enrichment transformations (count)",
		},
		[]string{"rule_id", "rule_name", "status"},
	)

	EnrichmentProviderRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_provider_requests_total",
			Help: "Total number of requests to enrichment providers (count)",
		},
		[]string{"provider", "status"},
	)

	EnrichmentProviderDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "enrichment_provider_duration_ms",
			Help:    "Duration of enrichment provider requests in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"provider"},
	)

	DatabaseQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries (count)",
		},
		[]string{"service", "database", "operation", "status"},
	)

	DatabaseQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_ms",
			Help:    "Duration of database queries in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"service", "database", "operation"},
	)

	DatabaseConnectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections (count)",
		},
		[]string{"service", "database"},
	)

	MessageQueueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "message_queue_size",
			Help: "Current size of message processing queue (count)",
		},
		[]string{"service"},
	)

	MessageQueueWaitDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "message_queue_wait_duration_ms",
			Help:    "Duration messages wait in queue before processing in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"service"},
	)
)

func RegisterFilteringMetrics() {
	prometheus.MustRegister(FilteringMessagesTotal)
	prometheus.MustRegister(FilteringProcessingDuration)
	prometheus.MustRegister(FilteringActiveRules)
	prometheus.MustRegister(FilteringRuleEvaluationsTotal)
	registerFallbackUsageTotalOnce()
}

func RegisterDedupMetrics() {
	prometheus.MustRegister(DeduplicateMessagesTotal)
	prometheus.MustRegister(DedupProcessingDuration)
	prometheus.MustRegister(DedupCacheSize)
	registerFallbackUsageTotalOnce()
}

func RegisterEnrichmentMetrics() {
	prometheus.MustRegister(EnrichmentMessagesTotal)
	prometheus.MustRegister(EnrichmentProcessingDuration)
	prometheus.MustRegister(EnrichmentActiveRules)
	prometheus.MustRegister(EnrichmentCacheHitRate)
	prometheus.MustRegister(EnrichmentRuleApplicationsTotal)
	prometheus.MustRegister(EnrichmentTransformationsTotal)
	prometheus.MustRegister(EnrichmentProviderRequestsTotal)
	prometheus.MustRegister(EnrichmentProviderDuration)
	registerFallbackUsageTotalOnce()
}

func registerFallbackUsageTotalOnce() {
	prometheus.MustRegister(FallbackUsageTotal)
}

func RegisterBrokerMetrics() {
	prometheus.MustRegister(RetryAttemptsTotal)
	prometheus.MustRegister(DLQMessagesTotal)
	prometheus.MustRegister(KafkaMessagesReadTotal)
	prometheus.MustRegister(KafkaMessagesWrittenTotal)
	prometheus.MustRegister(KafkaMessageSizeBytes)
	prometheus.MustRegister(KafkaConsumerLag)
	prometheus.MustRegister(KafkaReadDuration)
	prometheus.MustRegister(KafkaWriteDuration)
}

func RegisterCircuitBreakerMetrics() {
	prometheus.MustRegister(CircuitBreakerState)
	prometheus.MustRegister(CircuitBreakerRequests)
	prometheus.MustRegister(CircuitBreakerFailures)
}

func RegisterManagementMetrics() {
	prometheus.MustRegister(RateLimitRequestsTotal)
	prometheus.MustRegister(DatabaseQueriesTotal)
	prometheus.MustRegister(DatabaseQueryDuration)
	prometheus.MustRegister(DatabaseConnectionsActive)
	prometheus.MustRegister(MessageQueueSize)
	prometheus.MustRegister(MessageQueueWaitDuration)
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

// Helper functions for new metrics
func IncKafkaMessagesRead(service, topic string) {
	KafkaMessagesReadTotal.WithLabelValues(service, topic).Inc()
}

func IncKafkaMessagesWritten(service, topic string) {
	KafkaMessagesWrittenTotal.WithLabelValues(service, topic).Inc()
}

func ObserveKafkaMessageSize(service, topic, direction string, sizeBytes int) {
	KafkaMessageSizeBytes.WithLabelValues(service, topic, direction).Observe(float64(sizeBytes))
}

func SetKafkaConsumerLag(service, topic string, partition int, lag int64) {
	KafkaConsumerLag.WithLabelValues(service, topic, fmt.Sprintf("%d", partition)).Set(float64(lag))
}

func ObserveKafkaReadDuration(service, topic string, duration time.Duration) {
	KafkaReadDuration.WithLabelValues(service, topic).Observe(float64(duration.Milliseconds()))
}

func ObserveKafkaWriteDuration(service, topic string, duration time.Duration) {
	KafkaWriteDuration.WithLabelValues(service, topic).Observe(float64(duration.Milliseconds()))
}

func IncFilteringRuleEvaluation(ruleID, ruleName, result string) {
	FilteringRuleEvaluationsTotal.WithLabelValues(ruleID, ruleName, result).Inc()
}

func IncEnrichmentRuleApplication(ruleID, ruleName, status string) {
	EnrichmentRuleApplicationsTotal.WithLabelValues(ruleID, ruleName, status).Inc()
}

func IncEnrichmentTransformation(ruleID, ruleName, status string) {
	EnrichmentTransformationsTotal.WithLabelValues(ruleID, ruleName, status).Inc()
}

func IncEnrichmentProviderRequest(provider, status string) {
	EnrichmentProviderRequestsTotal.WithLabelValues(provider, status).Inc()
}

func ObserveEnrichmentProviderDuration(provider string, duration time.Duration) {
	EnrichmentProviderDuration.WithLabelValues(provider).Observe(float64(duration.Milliseconds()))
}

func IncDatabaseQuery(service, database, operation, status string) {
	DatabaseQueriesTotal.WithLabelValues(service, database, operation, status).Inc()
}

func ObserveDatabaseQueryDuration(service, database, operation string, duration time.Duration) {
	DatabaseQueryDuration.WithLabelValues(service, database, operation).Observe(float64(duration.Milliseconds()))
}

func SetDatabaseConnectionsActive(service, database string, count int) {
	DatabaseConnectionsActive.WithLabelValues(service, database).Set(float64(count))
}

func SetMessageQueueSize(service string, size int) {
	MessageQueueSize.WithLabelValues(service).Set(float64(size))
}

func ObserveMessageQueueWaitDuration(service string, duration time.Duration) {
	MessageQueueWaitDuration.WithLabelValues(service).Observe(float64(duration.Milliseconds()))
}
