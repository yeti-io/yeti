package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"

	"yeti/internal/broker"
	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/filtering"
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
	db             *sql.DB
	service        *filtering.Service
	tracerProvider *tracing.TracerProvider
	server         *http.Server
}

func NewApp(cfg *config.Config, log logger.Logger) *App {
	if sugaredLogger, ok := log.(*logger.SugaredLogger); ok {
		sugaredLogger.SetServiceName("filtering-service")
	}
	return &App{
		Base:        bootstrap.NewBase(cfg, log),
		dbConnector: bootstrap.NewDatabaseConnector(cfg, log),
	}
}

func (a *App) Initialize(ctx context.Context) error {
	if err := a.initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := a.initService(ctx); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	if err := a.InitBroker("filtering-service"); err != nil {
		return fmt.Errorf("failed to initialize broker: %w", err)
	}

	tp, err := tracing.Init(a.Config.Tracing, "filtering-service")
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}
	a.tracerProvider = tp

	metrics.RegisterFilteringMetrics()
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

	healthRegistry := health.NewCheckerRegistry()
	if a.db != nil {
		healthRegistry.Register(health.NewPostgreSQLChecker(a.db))
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

	mux.Handle("/metrics", promhttp.Handler())

	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.Config.Server.Port),
		Handler: mux,
	}

	return nil
}

func (a *App) initDatabase(ctx context.Context) error {
	db, err := a.dbConnector.InitPostgreSQL(ctx)
	if err != nil {
		return err
	}
	a.db = db
	return nil
}

func (a *App) initService(ctx context.Context) error {
	repo := filtering.NewRepository(a.db)
	svc, err := filtering.NewService(repo, a.Config.Filtering, a.Logger)
	if err != nil {
		return fmt.Errorf("failed to create filtering service: %w", err)
	}

	if err := svc.ReloadRules(ctx); err != nil {
		initCtx := logging.WithServiceName(ctx, "filtering-service")
		a.Logger.WarnwCtx(initCtx, "Failed to load initial rules",
			"error", err,
		)
	}

	a.service = svc
	return nil
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	if a.server != nil {
		g.Go(func() error {
			a.Logger.InfowCtx(ctx, "HTTP server starting", "port", a.Config.Server.Port)
			if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("HTTP server error: %w", err)
			}
			return nil
		})
	}

	configConsumer, err := broker.NewConsumer(a.Config.Broker, a.Logger)
	if err != nil {
		configCtx := logging.WithServiceName(ctx, "filtering-service")
		a.Logger.WarnwCtx(configCtx, "Failed to create config event consumer, event-driven reload disabled",
			"error", err,
		)
	} else {
		configConsumer.SetServiceName("filtering-service")
		defer configConsumer.Close()
		configEventHandler := filtering.NewHandler(a.service, a.Logger)

		g.Go(func() error {
			configCtx := logging.WithServiceName(gCtx, "filtering-service")
			a.Logger.InfowCtx(configCtx, "Starting config update event consumer",
				"topic", a.Config.Broker.Kafka.ConfigUpdateTopic,
			)
			return configConsumer.Consume(gCtx, a.Config.Broker.Kafka.ConfigUpdateTopic, func(cCtx context.Context, msg models.MessageEnvelope) error {
				return configEventHandler.HandleConfigUpdateEvent(cCtx, msg)
			})
		})
	}

	g.Go(func() error {
		return a.service.StartReloader(gCtx)
	})

	inputTopic := a.Config.Broker.Kafka.InputTopic
	g.Go(func() error {
		return a.Consumer.Consume(gCtx, inputTopic, a.handleMessage)
	})

	return g.Wait()
}

func (a *App) handleMessage(ctx context.Context, msg models.MessageEnvelope) error {
	passed, appliedRules, err := a.service.Filter(ctx, msg)
	if err != nil {
		a.Logger.ErrorwCtx(ctx, "Filter error",
			"error", err,
		)
		return err
	}

	if !passed {
		a.Logger.InfowCtx(ctx, "Message filtered out")
		return nil
	}

	if msg.Metadata.FiltersApplied == nil {
		msg.Metadata.FiltersApplied = &models.FiltersApplied{}
	}
	msg.Metadata.FiltersApplied.PassedAt = time.Now()
	msg.Metadata.FiltersApplied.RuleIDs = appliedRules

	outputTopic := a.Config.Broker.Kafka.OutputTopic
	if outputTopic == "" {
		outputTopic = "dedup_events"
	}

	if err := a.Producer.Publish(ctx, outputTopic, msg); err != nil {
		a.Logger.ErrorwCtx(ctx, "Failed to publish message",
			"error", err,
			"output_topic", outputTopic,
		)
		return err
	}
	a.Logger.InfowCtx(ctx, "Message passed filtering",
		"rules_applied", len(appliedRules),
	)

	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx := logging.WithServiceName(ctx, "filtering-service")
	a.Logger.InfowCtx(shutdownCtx, "Shutting down filtering service")

	additionalShutdown := func(ctx context.Context) []error {
		var errs []error

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

		errs = append(errs, a.dbConnector.ShutdownDatabases(ctx, nil, a.db, nil)...)

		return errs
	}

	return a.Base.Shutdown(ctx, additionalShutdown)
}
