package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"yeti/internal/broker"
	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/deduplication"
	"yeti/internal/logger"
	"yeti/pkg/bootstrap"
	"yeti/pkg/health"
	"yeti/pkg/logging"
	"yeti/pkg/metrics"
	"yeti/pkg/models"
	"yeti/pkg/tracing"
)

type App struct {
	*bootstrap.Base
	dbConnector    *bootstrap.DatabaseConnector
	redis          *redis.Client
	service        *deduplication.Service
	tracerProvider *tracing.TracerProvider
	server         *http.Server
}

func NewApp(cfg *config.Config, log logger.Logger) *App {
	if sugaredLogger, ok := log.(*logger.SugaredLogger); ok {
		sugaredLogger.SetServiceName("dedup-service")
	}
	return &App{
		Base:        bootstrap.NewBase(cfg, log),
		dbConnector: bootstrap.NewDatabaseConnector(cfg, log),
	}
}

func (a *App) Initialize(ctx context.Context) error {
	if err := a.initRedis(ctx); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	if err := a.initService(); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	if err := a.InitBroker("dedup-service"); err != nil {
		return fmt.Errorf("failed to initialize broker: %w", err)
	}

	tp, err := tracing.Init(a.Config.Tracing, "dedup-service")
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}
	a.tracerProvider = tp

	metrics.RegisterDedupMetrics()
	metrics.RegisterBrokerMetrics()
	if a.Config.CircuitBreaker.Enabled {
		metrics.RegisterCircuitBreakerMetrics()
	}

	if err := a.initHTTPServer(ctx); err != nil {
		return fmt.Errorf("failed to initialize HTTP server: %w", err)
	}

	return nil
}

func (a *App) initHTTPServer(ctx context.Context) error {
	mux := http.NewServeMux()

	// Health check endpoint
	healthRegistry := health.NewCheckerRegistry()
	if a.redis != nil {
		healthRegistry.Register(health.NewRedisChecker(a.redis))
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		h := healthRegistry.Check(r.Context())
		statusCode := http.StatusOK
		if h.Status == health.StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, `{"status":"%s","timestamp":"%s"}`, h.Status, h.Timestamp.Format(time.RFC3339))
	})

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.Config.Server.Port),
		Handler: mux,
	}

	return nil
}

func (a *App) initRedis(ctx context.Context) error {
	rdb, err := a.dbConnector.InitRedis(ctx)
	if err != nil {
		return err
	}
	a.redis = rdb
	return nil
}

func (a *App) initService() error {
	baseRepo := deduplication.NewRepository(a.redis)

	var repo deduplication.Repository
	if a.Config.CircuitBreaker.Enabled {
		repo = deduplication.NewCircuitBreakerRepository(baseRepo, a.Config.CircuitBreaker)
		initCtx := logging.WithServiceName(context.Background(), "dedup-service")
		a.Logger.InfowCtx(initCtx, "Circuit breaker enabled for deduplication repository")
	} else {
		repo = baseRepo
	}

	svc := deduplication.NewService(repo, a.Config.Deduplication, a.Logger)
	a.service = svc
	return nil
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	// Start HTTP server
	if a.server != nil {
		g.Go(func() error {
			a.Logger.InfowCtx(ctx, "HTTP server starting", "port", a.Config.Server.Port)
			if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("HTTP server error: %w", err)
			}
			return nil
		})
	}

	if a.Config.Broker.Type == "kafka" && a.Config.Broker.Kafka.ConfigUpdateTopic != "" {
		configConsumer, err := broker.NewConsumer(a.Config.Broker, a.Logger)
		if err != nil {
			configCtx := logging.WithServiceName(ctx, "dedup-service")
			a.Logger.WarnwCtx(configCtx, "Failed to create config event consumer, event-driven reload disabled",
				"error", err,
			)
		} else {
			configConsumer.SetServiceName("dedup-service")
			defer configConsumer.Close()
			configEventHandler := deduplication.NewHandler(a.service, a.Logger)

			g.Go(func() error {
				configCtx := logging.WithServiceName(gCtx, "dedup-service")
				a.Logger.InfowCtx(configCtx, "Starting config update event consumer",
					"topic", a.Config.Broker.Kafka.ConfigUpdateTopic,
				)
				return configConsumer.Consume(gCtx, a.Config.Broker.Kafka.ConfigUpdateTopic, func(cCtx context.Context, msg models.MessageEnvelope) error {
					return configEventHandler.HandleConfigUpdateEvent(cCtx, msg)
				})
			})
		}
	}

	inputTopic := a.Config.Broker.Kafka.InputTopic
	if inputTopic == "" {
		inputTopic = "filtered_events"
	}
	outputTopic := a.Config.Broker.Kafka.OutputTopic
	if outputTopic == "" {
		outputTopic = "deduplicated_events"
	}

	g.Go(func() error {
		return a.Consumer.Consume(gCtx, inputTopic, a.handleMessage(outputTopic))
	})

	return g.Wait()
}

func (a *App) handleMessage(outputTopic string) func(context.Context, models.MessageEnvelope) error {
	return func(ctx context.Context, msg models.MessageEnvelope) error {
		isUnique, err := a.service.Process(ctx, msg)
		if err != nil {
			a.Logger.ErrorwCtx(ctx, "Dedup processing error",
				"error", err,
			)
			return nil
		}

		if !isUnique {
			a.Logger.InfowCtx(ctx, "Message duplicate")
			return nil
		}
		if msg.Metadata.Deduplication == nil {
			msg.Metadata.Deduplication = &models.DeduplicationInfo{}
		}
		msg.Metadata.Deduplication.IsUnique = true
		msg.Metadata.Deduplication.CheckedAt = time.Now()

		if err := a.Producer.Publish(ctx, outputTopic, msg); err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}
		a.Logger.InfowCtx(ctx, "Message unique",
			"output_topic", outputTopic,
		)
		return nil
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx := logging.WithServiceName(ctx, "dedup-service")
	a.Logger.InfowCtx(shutdownCtx, "Shutting down deduplication service")

	additionalShutdown := func(ctx context.Context) []error {
		var errs []error

		if a.service != nil {
			a.service.StopCacheMetricsUpdater()
		}

		if a.server != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), constants.ShutdownTimeout)
			defer cancel()
			if err := a.server.Shutdown(shutdownCtx); err != nil {
				errs = append(errs, fmt.Errorf("HTTP server shutdown error: %w", err))
			}
		}

		if a.tracerProvider != nil {
			if err := a.tracerProvider.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("tracer provider shutdown error: %w", err))
			}
		}

		errs = append(errs, a.dbConnector.ShutdownDatabases(ctx, a.redis, nil, nil)...)

		return errs
	}

	return a.Base.Shutdown(ctx, additionalShutdown)
}
