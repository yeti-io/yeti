package config_handler

import (
	"context"
	"encoding/json"

	"yeti/internal/logger"
	"yeti/pkg/models"
)

type ConfigReloader interface {
	ReloadRules(ctx context.Context) error
}

type ConfigUpdater interface {
	UpdateFieldsToHash(fields []string) error
}

type Handler struct {
	expectedEventType   string
	expectedServiceType string
	reloader            ConfigReloader
	updater             ConfigUpdater
	logger              logger.Logger
}

func NewHandler(expectedEventType, expectedServiceType string, log logger.Logger) *Handler {
	return &Handler{
		expectedEventType:   expectedEventType,
		expectedServiceType: expectedServiceType,
		logger:              log,
	}
}

func NewHandlerWithReloader(expectedEventType, expectedServiceType string, reloader ConfigReloader, log logger.Logger) *Handler {
	return NewHandler(expectedEventType, expectedServiceType, log).WithReloader(reloader)
}

func NewHandlerWithUpdater(expectedEventType, expectedServiceType string, updater ConfigUpdater, log logger.Logger) *Handler {
	return NewHandler(expectedEventType, expectedServiceType, log).WithUpdater(updater)
}

func (h *Handler) WithReloader(reloader ConfigReloader) *Handler {
	h.reloader = reloader
	return h
}

func (h *Handler) WithUpdater(updater ConfigUpdater) *Handler {
	h.updater = updater
	return h
}

func (h *Handler) HandleConfigUpdateEvent(ctx context.Context, envelope models.MessageEnvelope) error {
	eventType, ok := envelope.Metadata.Enrichment["event_type"].(string)
	if !ok {
		if eventTypeVal, ok := envelope.Payload["event_type"].(string); ok {
			eventType = eventTypeVal
		} else {
			h.logger.Warnw("Config event missing event_type", "id", envelope.ID)
			return nil
		}
	}

	if eventType != h.expectedEventType {
		return nil
	}

	serviceType, ok := envelope.Metadata.Enrichment["service_type"].(string)
	if !ok {
		if serviceTypeVal, ok := envelope.Payload["service_type"].(string); ok {
			serviceType = serviceTypeVal
		} else {
			h.logger.Warnw("Config event missing service_type", "id", envelope.ID)
			return nil
		}
	}

	if serviceType != h.expectedServiceType {
		return nil
	}

	var event models.ConfigUpdateEvent
	eventJSON, err := json.Marshal(envelope.Payload)
	if err != nil {
		h.logger.Errorw("Failed to marshal event payload", "error", err, "id", envelope.ID)
		return err
	}

	if err := json.Unmarshal(eventJSON, &event); err != nil {
		h.logger.Errorw("Failed to unmarshal config event", "error", err, "id", envelope.ID)
		return err
	}

	h.logger.Infow("Received config update event",
		"event_type", event.EventType,
		"action", event.Action,
		"rule_id", event.RuleID,
	)

	if h.reloader != nil {
		if err := h.reloader.ReloadRules(ctx); err != nil {
			h.logger.Errorw("Failed to reload rules after config update", "error", err)
			return err
		}
		h.logger.Infow("Rules reloaded successfully after config update", "action", event.Action)
	}

	if h.updater == nil {
		return nil
	}

	if fields, ok := envelope.Payload["fields_to_hash"].([]interface{}); ok {
		fieldsStr := make([]string, len(fields))
		for i, f := range fields {
			if str, ok := f.(string); ok {
				fieldsStr[i] = str
			}
		}
		if len(fieldsStr) > 0 {
			if err := h.updater.UpdateFieldsToHash(fieldsStr); err != nil {
				h.logger.Errorw("Failed to update fields to hash", "error", err)
				return err
			}
		}
	}

	return nil
}
