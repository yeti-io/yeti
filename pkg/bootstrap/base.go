package bootstrap

import (
	"context"
	"fmt"

	"yeti/internal/broker"
	"yeti/internal/config"
	"yeti/internal/logger"
)

type Base struct {
	Config   *config.Config
	Logger   logger.Logger
	Producer broker.Producer
	Consumer broker.Consumer
}

func NewBase(cfg *config.Config, log logger.Logger) *Base {
	return &Base{
		Config: cfg,
		Logger: log,
	}
}

func (b *Base) InitBroker(serviceName string) error {
	producer, err := broker.NewProducer(b.Config.Broker, b.Logger)
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}

	consumer, err := broker.NewConsumer(b.Config.Broker, b.Logger)
	if err != nil {
		producer.Close()
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	if serviceName != "" {
		consumer.SetServiceName(serviceName)
	}

	b.Producer = producer
	b.Consumer = consumer
	return nil
}

func (b *Base) ShutdownBroker() []error {
	var errs []error

	if b.Producer != nil {
		if err := b.Producer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("producer close error: %w", err))
		}
	}

	if b.Consumer != nil {
		if err := b.Consumer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("consumer close error: %w", err))
		}
	}

	return errs
}

func (b *Base) Shutdown(ctx context.Context, additionalShutdown func(ctx context.Context) []error) error {
	b.Logger.Info("Shutting down application...")

	var errs []error

	errs = append(errs, b.ShutdownBroker()...)

	if additionalShutdown != nil {
		errs = append(errs, additionalShutdown(ctx)...)
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	b.Logger.Info("Application exited successfully")
	return nil
}
