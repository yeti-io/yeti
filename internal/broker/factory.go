package broker

import (
	"fmt"
	"yeti/internal/config"
	"yeti/internal/logger"
)

func NewProducer(cfg config.BrokerConfig, log logger.Logger) (Producer, error) {
	switch cfg.Type {
	case "kafka":
		return NewKafkaProducer(cfg.Kafka, log), nil
	default:
		return nil, fmt.Errorf("unknown broker type: %s", cfg.Type)
	}
}

func NewConsumer(cfg config.BrokerConfig, log logger.Logger) (Consumer, error) {
	switch cfg.Type {
	case "kafka":
		return NewKafkaConsumer(cfg.Kafka, log), nil
	default:
		return nil, fmt.Errorf("unknown broker type: %s", cfg.Type)
	}
}
