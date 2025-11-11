package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfig(configFile string) (*Config, error) {
	viper.Reset()

	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFile)

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	bindEnvVariables()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := applyEnvOverrides(&cfg); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	if err := ValidateStatic(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

func bindEnvVariables() {
	viper.BindEnv("broker.kafka.brokers", "BROKER_KAFKA_BROKERS")
	viper.BindEnv("broker.kafka.group_id", "BROKER_KAFKA_GROUP_ID")
	viper.BindEnv("broker.kafka.input_topic", "BROKER_KAFKA_INPUT_TOPIC")
	viper.BindEnv("broker.kafka.output_topic", "BROKER_KAFKA_OUTPUT_TOPIC")
	viper.BindEnv("broker.kafka.config_update_topic", "BROKER_KAFKA_CONFIG_UPDATE_TOPIC")
	viper.BindEnv("broker.kafka.dlq_topic", "BROKER_KAFKA_DLQ_TOPIC")

	viper.BindEnv("database.postgres.host", "DATABASE_POSTGRES_HOST")
	viper.BindEnv("database.postgres.port", "DATABASE_POSTGRES_PORT")
	viper.BindEnv("database.postgres.user", "DATABASE_POSTGRES_USER")
	viper.BindEnv("database.postgres.password", "DATABASE_POSTGRES_PASSWORD")
	viper.BindEnv("database.postgres.dbname", "DATABASE_POSTGRES_DBNAME")
	viper.BindEnv("database.postgres.sslmode", "DATABASE_POSTGRES_SSLMODE")

	viper.BindEnv("database.redis.host", "DATABASE_REDIS_HOST")
	viper.BindEnv("database.redis.port", "DATABASE_REDIS_PORT")
	viper.BindEnv("database.redis.password", "DATABASE_REDIS_PASSWORD")
	viper.BindEnv("database.redis.db", "DATABASE_REDIS_DB")

	viper.BindEnv("database.mongodb.uri", "DATABASE_MONGODB_URI")
	viper.BindEnv("database.mongodb.database", "DATABASE_MONGODB_DATABASE")

	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.read_timeout_seconds", "SERVER_READ_TIMEOUT_SECONDS")
	viper.BindEnv("server.write_timeout_seconds", "SERVER_WRITE_TIMEOUT_SECONDS")

	viper.BindEnv("logging.level", "LOGGING_LEVEL")
	viper.BindEnv("logging.format", "LOGGING_FORMAT")

	viper.BindEnv("tracing.otlp.endpoint", "TRACING_OTLP_ENDPOINT")
	viper.BindEnv("tracing.otlp.insecure", "TRACING_OTLP_INSECURE")
	viper.BindEnv("tracing.enabled", "TRACING_ENABLED")
	viper.BindEnv("tracing.service_name", "TRACING_SERVICE_NAME")
}

func applyEnvOverrides(cfg *Config) error {
	if brokersEnv := viper.GetString("BROKER_KAFKA_BROKERS"); brokersEnv != "" {
		brokers := strings.Split(brokersEnv, ",")
		for i := range brokers {
			brokers[i] = strings.TrimSpace(brokers[i])
		}
		if len(brokers) > 0 && brokers[0] != "" {
			cfg.Broker.Kafka.Brokers = brokers
		}
	}

	if otlpEndpoint := viper.GetString("TRACING_OTLP_ENDPOINT"); otlpEndpoint != "" {
		cfg.Tracing.OTLP.Endpoint = otlpEndpoint
	}

	return nil
}
