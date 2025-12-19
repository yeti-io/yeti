package config

import (
	"time"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Broker         BrokerConfig
	Logging        LoggingConfig
	Filtering      FilteringConfig
	Deduplication  DeduplicationConfig
	Enrichment     EnrichmentConfig
	Management     ManagementConfig
	CircuitBreaker CircuitBreakerConfig
	Tracing        TracingConfig
}

type DynamicConfig struct{}

type ServerConfig struct {
	Port                int           `mapstructure:"port"`
	ReadTimeoutSeconds  time.Duration `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds time.Duration `mapstructure:"write_timeout_seconds"`
}

type DatabaseConfig struct {
	Postgres      PostgresConfig
	Redis         RedisConfig
	MongoDB       MongoDBConfig
	RunMigrations bool `mapstructure:"run_migrations"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Password   string `mapstructure:"password"`
	DB         int    `mapstructure:"db"`
	TTLSeconds int    `mapstructure:"ttl_seconds"`
}

type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

type BrokerConfig struct {
	Type     string         `mapstructure:"type"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
}

type RabbitMQConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	InputQueue  string `mapstructure:"input_queue"`
	OutputQueue string `mapstructure:"output_queue"`
}

type KafkaConfig struct {
	Brokers           []string    `mapstructure:"brokers"`
	GroupID           string      `mapstructure:"group_id"`
	InputTopic        string      `mapstructure:"input_topic"`
	OutputTopic       string      `mapstructure:"output_topic"`
	ConfigUpdateTopic string      `mapstructure:"config_update_topic"`
	DLQTopic          string      `mapstructure:"dlq_topic"`
	Retry             RetryConfig `mapstructure:"retry"`
}

type RetryConfig struct {
	MaxAttempts     int           `mapstructure:"max_attempts"`
	InitialInterval time.Duration `mapstructure:"initial_interval"`
	MaxInterval     time.Duration `mapstructure:"max_interval"`
	Multiplier      float64       `mapstructure:"multiplier"`
	MaxElapsedTime  time.Duration `mapstructure:"max_elapsed_time"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type FilteringConfig struct {
	Reload   ReloadConfig   `mapstructure:"reload"`
	Fallback FallbackConfig `mapstructure:"fallback"`
}

type FallbackConfig struct {
	OnError string `mapstructure:"on_error"` // "allow", "deny", "error" (default: "error")
}

type ReloadConfig struct {
	IntervalSeconds int `mapstructure:"interval_seconds"`
}

type DeduplicationConfig struct {
	HashAlgorithm string   `mapstructure:"hash_algorithm"`
	TTLSeconds    int      `mapstructure:"ttl_seconds"`
	OnRedisError  string   `mapstructure:"on_redis_error"`
	FieldsToHash  []string `mapstructure:"fields_to_hash"`
}

type EnrichmentConfig struct{}

type ManagementConfig struct {
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type RateLimitConfig struct {
	Enabled         bool    `mapstructure:"enabled"`
	RPS             float64 `mapstructure:"rps"`
	Burst           int     `mapstructure:"burst"`
	CleanupInterval int     `mapstructure:"cleanup_interval"`
	MaxAge          int     `mapstructure:"max_age"`
}

type CircuitBreakerConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	MaxRequests  uint32        `mapstructure:"max_requests"`
	Interval     time.Duration `mapstructure:"interval"`
	Timeout      time.Duration `mapstructure:"timeout"`
	FailureRatio float64       `mapstructure:"failure_ratio"`
	MinRequests  uint32        `mapstructure:"min_requests"`
}

type TracingConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	ServiceName string        `mapstructure:"service_name"`
	OTLP        OTLPConfig    `mapstructure:"otlp"`
	Sampler     SamplerConfig `mapstructure:"sampler"`
}

type OTLPConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Insecure bool   `mapstructure:"insecure"`
}

type SamplerConfig struct {
	Type  string  `mapstructure:"type"`
	Param float64 `mapstructure:"param"`
}

func Load(configFile string) (*Config, error) {
	return LoadConfig(configFile)
}
