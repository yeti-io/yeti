package logging

import (
	"context"
)

const (
	TraceIDKey     = "trace_id"
	MessageIDKey   = "message_id"
	ServiceNameKey = "service_name"
)

func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

func WithMessageID(ctx context.Context, messageID string) context.Context {
	return context.WithValue(ctx, MessageIDKey, messageID)
}

func WithServiceName(ctx context.Context, serviceName string) context.Context {
	return context.WithValue(ctx, ServiceNameKey, serviceName)
}

func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

func GetMessageID(ctx context.Context) string {
	if messageID, ok := ctx.Value(MessageIDKey).(string); ok {
		return messageID
	}
	return ""
}

func GetServiceName(ctx context.Context) string {
	if serviceName, ok := ctx.Value(ServiceNameKey).(string); ok {
		return serviceName
	}
	return ""
}

func GetLogFields(ctx context.Context) []interface{} {
	fields := make([]interface{}, 0, 6)

	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, "trace_id", traceID)
	}

	if messageID := GetMessageID(ctx); messageID != "" {
		fields = append(fields, "message_id", messageID)
	}

	if serviceName := GetServiceName(ctx); serviceName != "" {
		fields = append(fields, "service_name", serviceName)
	}

	return fields
}
