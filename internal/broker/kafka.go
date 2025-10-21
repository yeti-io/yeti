package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/logger"
	"yeti/pkg/errors"
	"yeti/pkg/logging"
	"yeti/pkg/metrics"
	"yeti/pkg/models"
	"yeti/pkg/retry"
	"yeti/pkg/tracing"
)

type KafkaProducer struct {
	writer *kafka.Writer
	logger logger.Logger
}

func NewKafkaProducer(cfg config.KafkaConfig, log logger.Logger) *KafkaProducer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: constants.KafkaBatchTimeout,
		WriteTimeout: constants.KafkaWriteTimeout,
		Async:        false,
	}
	return &KafkaProducer{writer: w, logger: log}
}

func (p *KafkaProducer) Publish(ctx context.Context, topic string, msg models.MessageEnvelope) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Inject trace context into Kafka headers
	headers := []kafka.Header{}
	headers = tracing.InjectTraceContext(ctx, headers)

	err = p.writer.WriteMessages(ctx,
		kafka.Message{
			Topic:   topic,
			Key:     []byte(msg.ID),
			Value:   body,
			Headers: headers,
			Time:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to write kafka message: %w", err)
	}

	return nil
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

type KafkaConsumer struct {
	cfg         config.KafkaConfig
	wg          sync.WaitGroup
	reader      *kafka.Reader
	logger      logger.Logger
	dlqProducer Producer
	serviceName string
}

func NewKafkaConsumer(cfg config.KafkaConfig, log logger.Logger) *KafkaConsumer {
	consumer := &KafkaConsumer{
		cfg:         cfg,
		logger:      log,
		serviceName: "unknown",
	}

	if cfg.DLQTopic != "" {
		consumer.dlqProducer = NewKafkaProducer(cfg, log)
	}

	return consumer
}

func (c *KafkaConsumer) SetServiceName(name string) {
	c.serviceName = name
}

func (c *KafkaConsumer) Consume(ctx context.Context, topic string, handler HandlerFunc) error {
		c.logger.Infow("Creating Kafka reader",
			"topic", topic,
			"brokers", c.cfg.Brokers,
			"group_id", c.cfg.GroupID,
			"service_name", c.serviceName,
		)

	c.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.cfg.Brokers,
		GroupID:  c.cfg.GroupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		consumeCtx := logging.WithServiceName(ctx, c.serviceName)
		c.logger.InfowCtx(consumeCtx, "Started consuming",
			"topic", topic,
		)

		for {
			m, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					c.logger.InfowCtx(consumeCtx, "Stopped consuming",
						"topic", topic,
						"reason", "context canceled",
					)
					return
				}
				c.logger.ErrorwCtx(consumeCtx, "Error fetching kafka message",
					"error", err,
					"topic", topic,
				)
				time.Sleep(time.Second)
				continue
			}

			var envelope models.MessageEnvelope
			if err := json.Unmarshal(m.Value, &envelope); err != nil {
				c.logger.ErrorwCtx(ctx, "Failed to unmarshal message",
					"error", err,
					"topic", topic,
					"service_name", c.serviceName,
				)
				_ = c.reader.CommitMessages(ctx, m)
				continue
			}

			// Extract trace context from Kafka headers and start span
			msgCtx, span := tracing.StartSpanFromKafkaMessage(ctx, "kafka.consume", m.Headers)
			defer span.End()

			// Enrich context with trace_id and message_id from envelope
			if envelope.Metadata.TraceID != "" {
				msgCtx = logging.WithTraceID(msgCtx, envelope.Metadata.TraceID)
			}
			msgCtx = logging.WithMessageID(msgCtx, envelope.ID)
			msgCtx = logging.WithServiceName(msgCtx, c.serviceName)

			if err := c.processMessageWithRetry(msgCtx, envelope, handler, topic); err != nil {
				c.logger.ErrorwCtx(msgCtx, "Failed to process message after retries",
					"error", err,
					"topic", topic,
				)
				if c.dlqProducer != nil && c.cfg.DLQTopic != "" {
					if dlqErr := c.sendToDLQ(msgCtx, envelope, err, topic); dlqErr != nil {
						c.logger.ErrorwCtx(msgCtx, "Failed to send message to DLQ",
							"error", dlqErr,
							"topic", topic,
						)
						_ = c.reader.CommitMessages(ctx, m)
					} else {
						_ = c.reader.CommitMessages(ctx, m)
					}
				} else {
					c.logger.WarnwCtx(msgCtx, "No DLQ configured, committing message to avoid blocking",
						"topic", topic,
					)
					_ = c.reader.CommitMessages(ctx, m)
				}
			} else {
				if err := c.reader.CommitMessages(ctx, m); err != nil {
					c.logger.ErrorwCtx(msgCtx, "Failed to commit message",
						"error", err,
						"topic", topic,
					)
				}
			}
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}

func (c *KafkaConsumer) Close() error {
	var err error
	if c.reader != nil {
		err = c.reader.Close()
	}
	if c.dlqProducer != nil {
		if closeErr := c.dlqProducer.Close(); closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}
	c.wg.Wait()
	return err
}

func (c *KafkaConsumer) processMessageWithRetry(ctx context.Context, envelope models.MessageEnvelope, handler HandlerFunc, topic string) error {
	policy := retry.Policy{
		MaxAttempts:     3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}

	if c.cfg.Retry.MaxAttempts > 0 {
		policy.MaxAttempts = c.cfg.Retry.MaxAttempts
	}
	if c.cfg.Retry.InitialInterval > 0 {
		policy.InitialInterval = c.cfg.Retry.InitialInterval
	}
	if c.cfg.Retry.MaxInterval > 0 {
		policy.MaxInterval = c.cfg.Retry.MaxInterval
	}
	if c.cfg.Retry.Multiplier > 0 {
		policy.Multiplier = c.cfg.Retry.Multiplier
	}
	if c.cfg.Retry.MaxElapsedTime > 0 {
		policy.MaxElapsedTime = c.cfg.Retry.MaxElapsedTime
	}

	return retry.RetryWithCallback(ctx, policy, func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = errors.RecoverPanic(r)
				c.logger.ErrorwCtx(ctx, "Panic recovered during message processing",
					"error", err,
					"topic", topic,
				)
			}
		}()
		return handler(ctx, envelope)
	}, func(attempt int, err error, nextDelay time.Duration) {
		metrics.RetryAttemptsTotal.WithLabelValues(c.serviceName, topic).Inc()
		c.logger.WarnwCtx(ctx, "Retrying message processing",
			"attempt", attempt,
			"max_attempts", policy.MaxAttempts,
			"next_delay", nextDelay,
			"error", err,
			"topic", topic,
		)
	})
}

func (c *KafkaConsumer) sendToDLQ(ctx context.Context, envelope models.MessageEnvelope, originalErr error, sourceTopic string) error {
	if envelope.Metadata.Enrichment == nil {
		envelope.Metadata.Enrichment = make(map[string]interface{})
	}
	envelope.Metadata.Enrichment["dlq_reason"] = originalErr.Error()
	envelope.Metadata.Enrichment["dlq_source_topic"] = sourceTopic
	envelope.Metadata.Enrichment["dlq_timestamp"] = time.Now()

	err := c.dlqProducer.Publish(ctx, c.cfg.DLQTopic, envelope)
	if err != nil {
		return fmt.Errorf("failed to publish to DLQ: %w", err)
	}

	metrics.DLQMessagesTotal.WithLabelValues(c.serviceName, sourceTopic, "max_retries_exceeded").Inc()
	c.logger.InfowCtx(ctx, "Message sent to DLQ",
		"source_topic", sourceTopic,
		"dlq_topic", c.cfg.DLQTopic,
		"reason", originalErr.Error(),
	)

	return nil
}
