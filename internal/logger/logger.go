package logger

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"yeti/pkg/logging"
)

type Logger interface {
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})
	Sync() error

	DebugwCtx(ctx context.Context, msg string, keysAndValues ...interface{})
	InfowCtx(ctx context.Context, msg string, keysAndValues ...interface{})
	WarnwCtx(ctx context.Context, msg string, keysAndValues ...interface{})
	ErrorwCtx(ctx context.Context, msg string, keysAndValues ...interface{})
}

type SugaredLogger struct {
	*zap.SugaredLogger
	serviceName string
}

func (l *SugaredLogger) SetServiceName(name string) {
	l.serviceName = name
}

func New(level string) (Logger, error) {
	cfg := zap.NewProductionConfig()

	cfg.Encoding = "json"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.EncoderConfig.MessageKey = "message"
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.StacktraceKey = "stacktrace"

	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	zapLogger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return &SugaredLogger{
		SugaredLogger: zapLogger.Sugar(),
	}, nil
}

func (l *SugaredLogger) DebugwCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	fields := l.getContextFields(ctx)
	l.Debugw(msg, append(fields, keysAndValues...)...)
}

func (l *SugaredLogger) InfowCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	fields := l.getContextFields(ctx)
	l.Infow(msg, append(fields, keysAndValues...)...)
}

func (l *SugaredLogger) WarnwCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	fields := l.getContextFields(ctx)
	l.Warnw(msg, append(fields, keysAndValues...)...)
}

func (l *SugaredLogger) ErrorwCtx(ctx context.Context, msg string, keysAndValues ...interface{}) {
	fields := l.getContextFields(ctx)
	l.Errorw(msg, append(fields, keysAndValues...)...)
}

func (l *SugaredLogger) getContextFields(ctx context.Context) []interface{} {
	fields := logging.GetLogFields(ctx)

	if l.serviceName != "" && logging.GetServiceName(ctx) == "" {
		fields = append(fields, "service_name", l.serviceName)
	}

	return fields
}

func NopLogger() Logger {
	return &SugaredLogger{
		SugaredLogger: zap.NewNop().Sugar(),
		serviceName:   "",
	}
}
