package integration

import (
	"time"

	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/logger"
	"yeti/internal/management"
	"yeti/pkg/models"
)

const (
	containerStartupTimeout = 60
	timestampDelay          = 10 * time.Millisecond
)

func createTestLogger() logger.Logger {
	return logger.NopLogger()
}

func createTestFilteringConfig() config.FilteringConfig {
	return config.FilteringConfig{
		Fallback: config.FallbackConfig{
			OnError: constants.FallbackAllow,
		},
		Reload: config.ReloadConfig{
			IntervalSeconds: 60,
		},
	}
}

func createTestDeduplicationConfig() config.DeduplicationConfig {
	return createTestDeduplicationConfigWithFields([]string{"id", "source"})
}

func createTestDeduplicationConfigWithFields(fields []string) config.DeduplicationConfig {
	return config.DeduplicationConfig{
		HashAlgorithm: "md5",
		TTLSeconds:    300,
		OnRedisError:  constants.FallbackAllow,
		FieldsToHash:  fields,
	}
}

func createTestFilteringRule(name, expression string, priority int, enabled bool) *management.FilteringRule {
	return &management.FilteringRule{
		Name:       name,
		Expression: expression,
		Priority:   priority,
		Enabled:    enabled,
	}
}

func createTestMessage(id, source string, payload map[string]interface{}) models.MessageEnvelope {
	return models.MessageEnvelope{
		ID:       id,
		Source:   source,
		Payload:  payload,
		Metadata: models.Metadata{},
	}
}
