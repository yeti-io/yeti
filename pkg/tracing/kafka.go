package tracing

import (
	"context"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceParentHeader = "traceparent"
	traceStateHeader  = "tracestate"
)

func InjectTraceContext(ctx context.Context, headers []kafka.Header) []kafka.Header {
	propagator := otel.GetTextMapPropagator()
	if propagator == nil {
		return headers
	}

	carrier := kafkaHeaderCarrier{headers: headers}
	propagator.Inject(ctx, carrier)

	return carrier.headers
}

func ExtractTraceContext(ctx context.Context, headers []kafka.Header) context.Context {
	propagator := otel.GetTextMapPropagator()
	if propagator == nil {
		return ctx
	}

	carrier := kafkaHeaderCarrier{headers: headers}
	return propagator.Extract(ctx, carrier)
}

type kafkaHeaderCarrier struct {
	headers []kafka.Header
}

func (c kafkaHeaderCarrier) Get(key string) string {
	for _, h := range c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c kafkaHeaderCarrier) Set(key, value string) {
	for i, h := range c.headers {
		if h.Key == key {
			c.headers[i].Value = []byte(value)
			return
		}
	}
	c.headers = append(c.headers, kafka.Header{
		Key:   key,
		Value: []byte(value),
	})
}

func (c kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for _, h := range c.headers {
		keys = append(keys, h.Key)
	}
	return keys
}

func StartSpanFromKafkaMessage(ctx context.Context, operationName string, headers []kafka.Header) (context.Context, trace.Span) {
	ctx = ExtractTraceContext(ctx, headers)

	tracer := GetTracer("yeti-kafka")
	return tracer.Start(ctx, operationName)
}
