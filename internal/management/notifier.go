package management

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	kafka "yeti/internal/broker"
	"yeti/pkg/models"
)

type ConfigEventProducer struct {
	producer kafka.Producer
	topic    string
}

func NewConfigEventProducer(producer kafka.Producer, topic string) *ConfigEventProducer {
	return &ConfigEventProducer{
		producer: producer,
		topic:    topic,
	}
}

func (p *ConfigEventProducer) PublishFilteringRuleEvent(ctx context.Context, action, ruleID, changedBy string) error {
	event := models.ConfigUpdateEvent{
		EventType:   models.EventTypeFilteringRuleUpdated,
		ServiceType: models.ServiceTypeFiltering,
		RuleID:      ruleID,
		Action:      action,
		Timestamp:   time.Now(),
		ChangedBy:   changedBy,
	}
	return p.publishEvent(ctx, event)
}

func (p *ConfigEventProducer) PublishEnrichmentRuleEvent(ctx context.Context, action, ruleID, changedBy string) error {
	event := models.ConfigUpdateEvent{
		EventType:   models.EventTypeEnrichmentRuleUpdated,
		ServiceType: models.ServiceTypeEnrichment,
		RuleID:      ruleID,
		Action:      action,
		Timestamp:   time.Now(),
		ChangedBy:   changedBy,
	}
	return p.publishEvent(ctx, event)
}

func (p *ConfigEventProducer) PublishDedupConfigEvent(ctx context.Context, action, changedBy string, metadata map[string]interface{}) error {
	event := models.ConfigUpdateEvent{
		EventType:   models.EventTypeDedupConfigUpdated,
		ServiceType: models.ServiceTypeDeduplication,
		Action:      action,
		Timestamp:   time.Now(),
		ChangedBy:   changedBy,
		Metadata:    metadata,
	}
	return p.publishEvent(ctx, event)
}

func (p *ConfigEventProducer) publishEvent(ctx context.Context, event models.ConfigUpdateEvent) error {
	if p.producer == nil || p.topic == "" {
		return nil
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal config event: %w", err)
	}

	var eventData map[string]interface{}
	if err := json.Unmarshal(eventJSON, &eventData); err != nil {
		return fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	envelope := models.MessageEnvelope{
		ID:        uuid.New().String(),
		Source:    "management-service",
		Timestamp: time.Now(),
		Payload:   eventData,
		Metadata:  models.Metadata{},
	}

	if envelope.Metadata.Enrichment == nil {
		envelope.Metadata.Enrichment = make(map[string]interface{})
	}
	envelope.Metadata.Enrichment["event_type"] = event.EventType
	envelope.Metadata.Enrichment["service_type"] = event.ServiceType

	if event.Metadata != nil {
		envelope.Metadata.Enrichment["metadata"] = event.Metadata
	}

	return p.producer.Publish(ctx, p.topic, envelope)
}
