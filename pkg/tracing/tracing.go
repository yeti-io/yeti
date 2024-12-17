package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"yeti/internal/config"
)

type TracerProvider struct {
	tp *sdktrace.TracerProvider
}

func (tp *TracerProvider) Tracer(name string) trace.Tracer {
	return tp.tp.Tracer(name)
}

func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.tp != nil {
		return tp.tp.Shutdown(ctx)
	}
	return nil
}

func Init(cfg config.TracingConfig, serviceName string) (*TracerProvider, error) {
	if !cfg.Enabled {
		tp := sdktrace.NewTracerProvider()
		return &TracerProvider{tp: tp}, nil
	}

	if serviceName == "" {
		serviceName = cfg.ServiceName
	}
	if serviceName == "" {
		serviceName = "yeti-service"
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLP.Endpoint),
	}
	if cfg.OTLP.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	sampler := createSampler(cfg.Sampler)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{tp: tp}, nil
}

func createSampler(cfg config.SamplerConfig) sdktrace.Sampler {
	switch cfg.Type {
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(cfg.Param)
	case "parentbased_always_on":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	case "parentbased_traceidratio":
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Param))
	case "always_on":
		fallthrough
	default:
		return sdktrace.AlwaysSample()
	}
}

func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
