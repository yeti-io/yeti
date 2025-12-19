package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yeti/internal/management"
	"yeti/pkg/models"
)

const (
	kafkaBroker        = "localhost:29092"
	inputTopic         = "input_events"
	processedTopic     = "processed_events"
	messageWaitTimeout = 30 * time.Second
)

func TestPipelineEndToEnd(t *testing.T) {
	createReq := management.CreateFilteringRuleRequest{
		Name:       "e2e_test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
		Enabled:    boolPtr(true),
	}
	ruleID := createFilteringRule(t, createReq)
	defer deleteFilteringRule(t, ruleID)

	time.Sleep(3 * time.Second)

	testMessage := models.MessageEnvelope{
		ID:        uuid.New().String(),
		Source:    "e2e_test",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"status": "active",
			"type":   "test",
			"value":  100,
		},
		Metadata: models.Metadata{},
	}

	err := sendMessageToKafka(t, inputTopic, testMessage)
	require.NoError(t, err, "failed to send message to input topic")

	processedMessage := waitForProcessedMessage(t, testMessage.ID)
	require.NotNil(t, processedMessage, "message should be processed")

	assert.Equal(t, testMessage.ID, processedMessage.ID)
	assert.Equal(t, testMessage.Source, processedMessage.Source)
	assert.Equal(t, "active", processedMessage.Payload["status"])

	assert.NotNil(t, processedMessage.Metadata)
	assert.NotNil(t, processedMessage.Metadata.FiltersApplied, "FiltersApplied should be set by filtering service")
	assert.NotEmpty(t, processedMessage.Metadata.FiltersApplied.RuleIDs, "RuleIDs should not be empty - rule should be applied")
	assert.Contains(t, processedMessage.Metadata.FiltersApplied.RuleIDs, ruleID,
		"Rule ID should be in applied rules list")
	assert.True(t, processedMessage.Metadata.Deduplication.IsUnique)
}

func TestPipelineFiltering(t *testing.T) {
	createReq := management.CreateFilteringRuleRequest{
		Name:       "filter_test_rule",
		Expression: "payload.value > 50",
		Priority:   10,
		Enabled:    boolPtr(true),
	}
	ruleID := createFilteringRule(t, createReq)
	defer deleteFilteringRule(t, ruleID)

	time.Sleep(2 * time.Second)

	passingMessage := models.MessageEnvelope{
		ID:        uuid.New().String(),
		Source:    "filter_test",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"value": 100,
		},
		Metadata: models.Metadata{},
	}

	err := sendMessageToKafka(t, inputTopic, passingMessage)
	require.NoError(t, err)

	processedMessage := waitForProcessedMessage(t, passingMessage.ID)
	require.NotNil(t, processedMessage, "message with value 100 should pass filter")

	filteredMessage := models.MessageEnvelope{
		ID:        uuid.New().String(),
		Source:    "filter_test",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"value": 30,
		},
		Metadata: models.Metadata{},
	}

	err = sendMessageToKafka(t, inputTopic, filteredMessage)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	notProcessed := tryGetProcessedMessage(t, filteredMessage.ID)
	assert.Nil(t, notProcessed, "message with value 30 should be filtered out")
}

func TestPipelineDeduplication(t *testing.T) {
	updateReq := management.UpdateDeduplicationConfigRequest{
		HashAlgorithm: stringPtr("sha256"),
		TTLSeconds:    intPtr(3600),
		OnRedisError:  stringPtr("allow"),
		FieldsToHash:  &[]string{"id", "source"},
	}
	_ = updateDeduplicationConfig(t, updateReq)

	testMessage := models.MessageEnvelope{
		ID:        "dedup-test-123",
		Source:    "dedup_test",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"status": "active",
		},
		Metadata: models.Metadata{},
	}

	err := sendMessageToKafka(t, inputTopic, testMessage)
	require.NoError(t, err)

	firstProcessed := waitForProcessedMessage(t, testMessage.ID)
	require.NotNil(t, firstProcessed, "first message should be processed")
	if firstProcessed.Metadata.Deduplication != nil {
		assert.True(t, firstProcessed.Metadata.Deduplication.IsUnique, "first message should be unique")
	}

	time.Sleep(3 * time.Second)

	duplicateMessage := models.MessageEnvelope{
		ID:        "dedup-test-123",
		Source:    "dedup_test",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"status": "active",
		},
		Metadata: models.Metadata{},
	}

	err = sendMessageToKafka(t, inputTopic, duplicateMessage)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	duplicateProcessed := tryGetProcessedMessage(t, duplicateMessage.ID)
	assert.Nil(t, duplicateProcessed, "duplicate message should be dropped and not appear in processed_events")
}

func TestPipelineEnrichment(t *testing.T) {
	createReq := management.CreateEnrichmentRuleRequest{
		Name:          "enrichment_test_rule",
		FieldToEnrich: "enrichment_data",
		SourceType:    "redis",
		SourceConfig: management.EnrichmentSourceConfig{
			KeyPattern: "test:key",
		},
		Priority:      10,
		Enabled:       boolPtr(true),
		ErrorHandling: "skip_field",
	}
	ruleID := createEnrichmentRule(t, createReq)
	defer deleteEnrichmentRule(t, ruleID)

	filterReq := management.CreateFilteringRuleRequest{
		Name:       "allow_all_for_enrichment",
		Expression: "true",
		Priority:   5,
		Enabled:    boolPtr(true),
	}
	filterRuleID := createFilteringRule(t, filterReq)
	defer deleteFilteringRule(t, filterRuleID)

	testMessage := models.MessageEnvelope{
		ID:        uuid.New().String(),
		Source:    "enrichment_test",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"status": "active",
		},
		Metadata: models.Metadata{},
	}

	err := sendMessageToKafka(t, inputTopic, testMessage)
	require.NoError(t, err)

	processedMessage := waitForProcessedMessage(t, testMessage.ID)
	require.NotNil(t, processedMessage)

	assert.NotNil(t, processedMessage.Metadata)
	assert.NotEmpty(t, processedMessage.Metadata.Enrichment)
}

func TestPipelineMultipleMessages(t *testing.T) {
	createReq := management.CreateFilteringRuleRequest{
		Name:       "multi_msg_test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
		Enabled:    boolPtr(true),
	}
	ruleID := createFilteringRule(t, createReq)
	defer deleteFilteringRule(t, ruleID)

	time.Sleep(2 * time.Second)

	messages := []models.MessageEnvelope{
		{
			ID:        uuid.New().String(),
			Source:    "multi_test",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"status": "active",
				"index":  1,
			},
			Metadata: models.Metadata{},
		},
		{
			ID:        uuid.New().String(),
			Source:    "multi_test",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"status": "active",
				"index":  2,
			},
			Metadata: models.Metadata{},
		},
		{
			ID:        uuid.New().String(),
			Source:    "multi_test",
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"status": "inactive",
				"index":  3,
			},
			Metadata: models.Metadata{},
		},
	}

	for _, msg := range messages {
		err := sendMessageToKafka(t, inputTopic, msg)
		require.NoError(t, err)
	}

	msg1 := waitForProcessedMessage(t, messages[0].ID)
	assert.NotNil(t, msg1, "first message should be processed")

	msg2 := waitForProcessedMessage(t, messages[1].ID)
	assert.NotNil(t, msg2, "second message should be processed")

	time.Sleep(3 * time.Second)
	msg3 := tryGetProcessedMessage(t, messages[2].ID)
	assert.Nil(t, msg3, "third message should be filtered out")
}

func sendMessageToKafka(t *testing.T, topic string, message models.MessageEnvelope) error {
	t.Helper()

	writer := &kafka.Writer{
		Addr:         kafka.TCP("localhost:29092"),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
	}
	defer writer.Close()

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(message.ID),
			Value: body,
			Time:  time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func waitForProcessedMessage(t *testing.T, messageID string) *models.MessageEnvelope {
	t.Helper()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaBroker},
		Topic:          processedTopic,
		GroupID:        fmt.Sprintf("e2e-test-waiter-%s", uuid.New().String()),
		StartOffset:    kafka.FirstOffset,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		MaxWait:        2 * time.Second,
	})
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), messageWaitTimeout)
	defer cancel()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				return nil
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var envelope models.MessageEnvelope
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		_ = reader.CommitMessages(ctx, msg)

		if envelope.ID == messageID {
			return &envelope
		}
	}
}

func tryGetProcessedMessage(t *testing.T, messageID string) *models.MessageEnvelope {
	t.Helper()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kafkaBroker},
		Topic:          processedTopic,
		GroupID:        fmt.Sprintf("e2e-test-reader-%s", uuid.New().String()),
		StartOffset:    kafka.LastOffset,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		MaxWait:        2 * time.Second,
	})
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				break
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var envelope models.MessageEnvelope
		if err := json.Unmarshal(msg.Value, &envelope); err != nil {
			_ = reader.CommitMessages(ctx, msg)
			continue
		}

		_ = reader.CommitMessages(ctx, msg)

		if envelope.ID == messageID {
			return &envelope
		}
	}

	return nil
}

func TestPipelineWithRuleUpdate(t *testing.T) {
	createReq := management.CreateFilteringRuleRequest{
		Name:       "update_test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
		Enabled:    boolPtr(true),
	}
	ruleID := createFilteringRule(t, createReq)
	defer deleteFilteringRule(t, ruleID)

	time.Sleep(2 * time.Second)

	msg1 := models.MessageEnvelope{
		ID:        uuid.New().String(),
		Source:    "update_test",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"status": "active",
		},
		Metadata: models.Metadata{},
	}

	err := sendMessageToKafka(t, inputTopic, msg1)
	require.NoError(t, err)

	processed1 := waitForProcessedMessage(t, msg1.ID)
	require.NotNil(t, processed1, "message should pass with initial rule")

	updateReq := management.UpdateFilteringRuleRequest{
		Expression: stringPtr("payload.status == 'inactive'"),
	}
	updatedRule := updateFilteringRule(t, ruleID, updateReq)
	assert.Equal(t, "payload.status == 'inactive'", updatedRule.Expression, "Rule expression should be updated")

	time.Sleep(10 * time.Second)

	msg2 := models.MessageEnvelope{
		ID:        uuid.New().String(),
		Source:    "update_test",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"status": "active",
		},
		Metadata: models.Metadata{},
	}

	err = sendMessageToKafka(t, inputTopic, msg2)
	require.NoError(t, err)

	processed2 := waitForProcessedMessage(t, msg2.ID)
	assert.Nil(t, processed2, "Message with status 'active' should be filtered out after rule update to filter 'inactive' (hot reload should work)")
}
