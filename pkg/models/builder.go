package models

import "time"

type MessageEnvelopeBuilder struct {
	envelope *MessageEnvelope
}

func NewMessageEnvelopeBuilder() *MessageEnvelopeBuilder {
	return &MessageEnvelopeBuilder{
		envelope: &MessageEnvelope{
			Payload:  make(map[string]interface{}),
			Metadata: Metadata{},
		},
	}
}

func (b *MessageEnvelopeBuilder) WithID(id string) *MessageEnvelopeBuilder {
	b.envelope.ID = id
	return b
}

func (b *MessageEnvelopeBuilder) WithSource(source string) *MessageEnvelopeBuilder {
	b.envelope.Source = source
	return b
}

func (b *MessageEnvelopeBuilder) WithTimestamp(timestamp time.Time) *MessageEnvelopeBuilder {
	b.envelope.Timestamp = timestamp
	return b
}

func (b *MessageEnvelopeBuilder) WithPayload(payload map[string]interface{}) *MessageEnvelopeBuilder {
	b.envelope.Payload = payload
	return b
}

func (b *MessageEnvelopeBuilder) WithMetadata(metadata Metadata) *MessageEnvelopeBuilder {
	b.envelope.Metadata = metadata
	return b
}

func (b *MessageEnvelopeBuilder) WithTraceID(traceID string) *MessageEnvelopeBuilder {
	b.envelope.Metadata.TraceID = traceID
	return b
}

func (b *MessageEnvelopeBuilder) Build() *MessageEnvelope {
	if b.envelope.Timestamp.IsZero() {
		b.envelope.Timestamp = time.Now()
	}
	return b.envelope
}
