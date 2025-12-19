package models

import "fmt"

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

func ValidateMessageEnvelope(msg *MessageEnvelope) error {
	if msg == nil {
		return &ValidationError{
			Field:   "envelope",
			Message: "message envelope cannot be nil",
		}
	}

	if msg.ID == "" {
		return &ValidationError{
			Field:   "id",
			Message: "message ID is required",
		}
	}

	if msg.Source == "" {
		return &ValidationError{
			Field:   "source",
			Message: "message source is required",
		}
	}

	if msg.Timestamp.IsZero() {
		return &ValidationError{
			Field:   "timestamp",
			Message: "message timestamp is required",
		}
	}

	if msg.Payload == nil {
		return &ValidationError{
			Field:   "payload",
			Message: "message payload cannot be nil",
		}
	}

	return nil
}

func (msg *MessageEnvelope) GetPayloadField(name string) (interface{}, bool) {
	if msg.Payload == nil {
		return nil, false
	}

	value, ok := msg.Payload[name]
	return value, ok
}

func (msg *MessageEnvelope) SetPayloadField(name string, value interface{}) {
	if msg.Payload == nil {
		msg.Payload = make(map[string]interface{})
	}

	msg.Payload[name] = value
}
