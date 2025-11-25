package broker

import (
	"context"
	"yeti/pkg/models"
)

type Producer interface {
	Publish(ctx context.Context, topic string, msg models.MessageEnvelope) error
	Close() error
}

type Consumer interface {
	Consume(ctx context.Context, topic string, handler HandlerFunc) error
	Close() error
	SetServiceName(name string)
}

type HandlerFunc func(ctx context.Context, msg models.MessageEnvelope) error
