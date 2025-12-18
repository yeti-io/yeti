package constants

import "time"

const (
	KafkaBatchTimeout = 10 * time.Millisecond
	KafkaWriteTimeout = 10 * time.Second
)

const (
	DefaultHTTPTimeout = 10 * time.Second
)

const (
	CacheKeyPrefixDedup  = "dedup:"
	CacheKeyPrefixEnrich = "enrich:"
)

const (
	DefaultInputTopic  = "deduplicated_events"
	DefaultOutputTopic = "processed_events"
)

const (
	DefaultMongoDBName = "yeti"
)

const (
	ShutdownTimeout = 5 * time.Second
)

const (
	DefaultLimit       = 100
	MaxLimit           = 1000
	DefaultTruncateLen = 100
)

const (
	DefaultTTLSeconds = 3600
)

const (
	HTTPStatusOKMin = 200
	HTTPStatusOKMax = 300
)

const (
	ErrorHandlingFail      = "fail"
	ErrorHandlingSkipRule  = "skip_rule"
	ErrorHandlingSkipField = "skip_field"
)

const (
	FallbackAllow = "allow"
	FallbackDeny  = "deny"
	FallbackError = "error"
)

const (
	SourceTypeAPI        = "api"
	SourceTypeDatabase   = "database"
	SourceTypeMongoDB    = "mongodb"
	SourceTypePostgreSQL = "postgresql"
	SourceTypeCache      = "cache"
	SourceTypeRedis      = "redis"
)

const (
	ProviderNameMongoDB    = "mongodb"
	ProviderNamePostgreSQL = "postgresql"
	ProviderNameCache      = "cache"
	ProviderNameAPI        = "api"
)
