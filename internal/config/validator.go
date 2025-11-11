package config

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

func ValidateStatic(cfg *Config) error {
	var errors []error

	if err := validateServer(cfg.Server); err != nil {
		errors = append(errors, err)
	}

	if err := validateBroker(cfg.Broker); err != nil {
		errors = append(errors, err)
	}

	if err := validateDatabase(cfg.Database); err != nil {
		errors = append(errors, err)
	}

	if err := validateDeduplication(cfg.Deduplication); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %v", errors)
	}

	return nil
}

func validateServer(cfg ServerConfig) error {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return &ValidationError{
			Field:   "server.port",
			Message: fmt.Sprintf("port must be between 1 and 65535, got %d", cfg.Port),
		}
	}

	if cfg.ReadTimeoutSeconds <= 0 {
		return &ValidationError{
			Field:   "server.read_timeout_seconds",
			Message: "read timeout must be positive",
		}
	}

	if cfg.WriteTimeoutSeconds <= 0 {
		return &ValidationError{
			Field:   "server.write_timeout_seconds",
			Message: "write timeout must be positive",
		}
	}

	return nil
}

func validateBroker(cfg BrokerConfig) error {
	if cfg.Type == "" {
		return &ValidationError{
			Field:   "broker.type",
			Message: "broker type is required",
		}
	}

	switch cfg.Type {
	case "kafka":
		return validateKafka(cfg.Kafka)
	case "rabbitmq":
		return validateRabbitMQ(cfg.RabbitMQ)
	default:
		return &ValidationError{
			Field:   "broker.type",
			Message: fmt.Sprintf("unknown broker type: %s (supported: kafka, rabbitmq)", cfg.Type),
		}
	}
}

func validateKafka(cfg KafkaConfig) error {
	if len(cfg.Brokers) == 0 {
		return &ValidationError{
			Field:   "broker.kafka.brokers",
			Message: "at least one Kafka broker is required",
		}
	}

	for i, broker := range cfg.Brokers {
		if broker == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("broker.kafka.brokers[%d]", i),
				Message: "broker address cannot be empty",
			}
		}
	}

	if cfg.GroupID == "" {
		return &ValidationError{
			Field:   "broker.kafka.group_id",
			Message: "Kafka consumer group ID is required",
		}
	}

	if cfg.Retry.MaxAttempts < 0 {
		return &ValidationError{
			Field:   "broker.kafka.retry.max_attempts",
			Message: "max_attempts must be non-negative",
		}
	}

	if cfg.Retry.InitialInterval < 0 {
		return &ValidationError{
			Field:   "broker.kafka.retry.initial_interval",
			Message: "initial_interval must be non-negative",
		}
	}

	if cfg.Retry.MaxInterval < 0 {
		return &ValidationError{
			Field:   "broker.kafka.retry.max_interval",
			Message: "max_interval must be non-negative",
		}
	}

	if cfg.Retry.MaxInterval > 0 && cfg.Retry.InitialInterval > 0 && cfg.Retry.MaxInterval < cfg.Retry.InitialInterval {
		return &ValidationError{
			Field:   "broker.kafka.retry.max_interval",
			Message: "max_interval must be greater than or equal to initial_interval",
		}
	}

	if cfg.Retry.Multiplier <= 0 {
		return &ValidationError{
			Field:   "broker.kafka.retry.multiplier",
			Message: "multiplier must be positive",
		}
	}

	return nil
}

func validateRabbitMQ(cfg RabbitMQConfig) error {
	if cfg.Host == "" {
		return &ValidationError{
			Field:   "broker.rabbitmq.host",
			Message: "RabbitMQ host is required",
		}
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return &ValidationError{
			Field:   "broker.rabbitmq.port",
			Message: fmt.Sprintf("port must be between 1 and 65535, got %d", cfg.Port),
		}
	}

	return nil
}

func validateDatabase(cfg DatabaseConfig) error {
	if cfg.Postgres.Host != "" || cfg.Postgres.Port > 0 {
		if err := validatePostgres(cfg.Postgres); err != nil {
			return err
		}
	}

	if cfg.Redis.Host != "" || cfg.Redis.Port > 0 {
		if err := validateRedis(cfg.Redis); err != nil {
			return err
		}
	}

	if cfg.MongoDB.URI != "" {
		if err := validateMongoDB(cfg.MongoDB); err != nil {
			return err
		}
	}

	return nil
}

func validatePostgres(cfg PostgresConfig) error {
	if cfg.Host == "" {
		return &ValidationError{
			Field:   "database.postgres.host",
			Message: "PostgreSQL host is required",
		}
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return &ValidationError{
			Field:   "database.postgres.port",
			Message: fmt.Sprintf("port must be between 1 and 65535, got %d", cfg.Port),
		}
	}

	if cfg.User == "" {
		return &ValidationError{
			Field:   "database.postgres.user",
			Message: "PostgreSQL user is required",
		}
	}

	if cfg.DBName == "" {
		return &ValidationError{
			Field:   "database.postgres.dbname",
			Message: "PostgreSQL database name is required",
		}
	}

	validSSLModes := map[string]bool{
		"disable": true, "allow": true, "prefer": true,
		"require": true, "verify-ca": true, "verify-full": true,
	}
	if cfg.SSLMode != "" && !validSSLModes[strings.ToLower(cfg.SSLMode)] {
		return &ValidationError{
			Field:   "database.postgres.sslmode",
			Message: fmt.Sprintf("invalid SSL mode: %s (valid: disable, allow, prefer, require, verify-ca, verify-full)", cfg.SSLMode),
		}
	}

	return nil
}

func validateRedis(cfg RedisConfig) error {
	if cfg.Host == "" {
		return &ValidationError{
			Field:   "database.redis.host",
			Message: "Redis host is required",
		}
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return &ValidationError{
			Field:   "database.redis.port",
			Message: fmt.Sprintf("port must be between 1 and 65535, got %d", cfg.Port),
		}
	}

	if cfg.TTLSeconds < 0 {
		return &ValidationError{
			Field:   "database.redis.ttl_seconds",
			Message: "TTL must be non-negative",
		}
	}

	return nil
}

func validateMongoDB(cfg MongoDBConfig) error {
	if cfg.URI == "" {
		return &ValidationError{
			Field:   "database.mongodb.uri",
			Message: "MongoDB URI is required",
		}
	}

	if !strings.HasPrefix(cfg.URI, "mongodb://") && !strings.HasPrefix(cfg.URI, "mongodb+srv://") {
		return &ValidationError{
			Field:   "database.mongodb.uri",
			Message: "MongoDB URI must start with mongodb:// or mongodb+srv://",
		}
	}

	if cfg.Database == "" {
		return &ValidationError{
			Field:   "database.mongodb.database",
			Message: "MongoDB database name is required",
		}
	}

	return nil
}

func validateDeduplication(cfg DeduplicationConfig) error {
	validAlgorithms := map[string]bool{
		"md5": true, "sha256": true, "sha1": true,
	}
	if cfg.HashAlgorithm != "" && !validAlgorithms[strings.ToLower(cfg.HashAlgorithm)] {
		return &ValidationError{
			Field:   "deduplication.hash_algorithm",
			Message: fmt.Sprintf("invalid hash algorithm: %s (valid: md5, sha256, sha1)", cfg.HashAlgorithm),
		}
	}

	if cfg.TTLSeconds < 0 {
		return &ValidationError{
			Field:   "deduplication.ttl_seconds",
			Message: "TTL must be non-negative",
		}
	}

	validOnError := map[string]bool{
		"allow": true, "reject": true, "fail": true,
	}
	if cfg.OnRedisError != "" && !validOnError[strings.ToLower(cfg.OnRedisError)] {
		return &ValidationError{
			Field:   "deduplication.on_redis_error",
			Message: fmt.Sprintf("invalid on_redis_error value: %s (valid: allow, reject, fail)", cfg.OnRedisError),
		}
	}

	return nil
}
