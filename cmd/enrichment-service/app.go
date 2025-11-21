package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/errgroup"

	"yeti/internal/broker"
	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/enrichment"
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
	mongoClient    *mongo.Client
	postgresDB     *sql.DB
	service        enrichment.Service
	tracerProvider *tracing.TracerProvider
	server         *http.Server
}

func NewApp(cfg *config.Config, log logger.Logger) *App {
	if sugaredLogger, ok := log.(*logger.SugaredLogger); ok {
		sugaredLogger.SetServiceName("enrichment-service")
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

	if err := a.initMongoDB(ctx); err != nil {
		return fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	if err := a.initPostgreSQL(ctx); err != nil {
		initCtx := logging.WithServiceName(ctx, "enrichment-service")
		a.Logger.WarnwCtx(initCtx, "PostgreSQL initialization failed, PostgreSQL provider will be disabled",
			"error", err,
		)
	}

	if err := a.initService(ctx); err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	if err := a.InitBroker("enrichment-service"); err != nil {
		return fmt.Errorf("failed to initialize broker: %w", err)
	}

	tp, err := tracing.Init(a.Config.Tracing, "enrichment-service")
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}
	a.tracerProvider = tp

	metrics.RegisterEnrichmentMetrics()
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
	if a.mongoClient != nil {
		healthRegistry.Register(health.NewMongoDBChecker(a.mongoClient))
	}
	if a.postgresDB != nil {
		healthRegistry.Register(health.NewPostgreSQLChecker(a.postgresDB))
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

func (a *App) initRedis(ctx context.Context) error {
	rdb, err := a.dbConnector.InitRedis(ctx)
	if err != nil {
		return err
	}
	a.redis = rdb
	return nil
}

func (a *App) initMongoDB(ctx context.Context) error {
	mongoClient, err := a.dbConnector.InitMongoDB(ctx)
	if err != nil {
		return err
	}

	if mongoClient != nil {
		a.mongoClient = mongoClient
	}
	return nil
}

func (a *App) initPostgreSQL(ctx context.Context) error {
	postgresDB, err := a.dbConnector.InitPostgreSQL(ctx)
	if err != nil {
		return err
	}
	if postgresDB != nil {
		a.postgresDB = postgresDB
	}
	return nil
}

func (a *App) initService(ctx context.Context) error {
	mongoDb := a.mongoClient.Database(a.Config.Database.MongoDB.Database)
	repo := enrichment.NewRepository(mongoDb)

	var svc enrichment.Service
	cbConfig := &a.Config.CircuitBreaker
	if a.mongoClient != nil || a.postgresDB != nil {
		svc = enrichment.NewServiceWithDatabaseProvidersAndCircuitBreaker(repo, a.redis, a.mongoClient, a.postgresDB, a.Logger, cbConfig)
	} else {
		svc = enrichment.NewServiceWithCircuitBreaker(repo, a.redis, a.Logger, cbConfig)
	}

	if err := svc.ReloadRules(ctx); err != nil {
		initCtx := logging.WithServiceName(ctx, "enrichment-service")
		a.Logger.WarnwCtx(initCtx, "Failed to load rules",
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

	if a.Config.Broker.Type == "kafka" && a.Config.Broker.Kafka.ConfigUpdateTopic != "" {
		configConsumer, err := broker.NewConsumer(a.Config.Broker, a.Logger)
		if err != nil {
			configCtx := logging.WithServiceName(ctx, "enrichment-service")
			a.Logger.WarnwCtx(configCtx, "Failed to create config event consumer, event-driven reload disabled",
				"error", err,
			)
		} else {
			configConsumer.SetServiceName("enrichment-service")
			defer configConsumer.Close()
			configEventHandler := enrichment.NewHandler(a.service, a.Logger)

			g.Go(func() error {
				configCtx := logging.WithServiceName(gCtx, "enrichment-service")
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
		inputTopic = constants.DefaultInputTopic
	}
	outputTopic := a.Config.Broker.Kafka.OutputTopic
	if outputTopic == "" {
		outputTopic = constants.DefaultOutputTopic
	}

	g.Go(func() error {
		return a.Consumer.Consume(gCtx, inputTopic, a.handleMessage(outputTopic))
	})

	return g.Wait()
}

func (a *App) handleMessage(outputTopic string) func(context.Context, models.MessageEnvelope) error {
	return func(ctx context.Context, msg models.MessageEnvelope) error {
		enrichedMsg, err := a.service.Process(ctx, msg)
		if err != nil {
			a.Logger.ErrorwCtx(ctx, "Enrichment error",
				"error", err,
			)
			return nil
		}

		if enrichedMsg.Metadata.Enrichment == nil {
			enrichedMsg.Metadata.Enrichment = make(map[string]interface{})
		}
		enrichedMsg.Metadata.Enrichment["processed_at"] = time.Now()

		if err := a.Producer.Publish(ctx, outputTopic, enrichedMsg); err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}

		a.Logger.InfowCtx(ctx, "Message enriched",
			"output_topic", outputTopic,
		)
		return nil
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	shutdownCtx := logging.WithServiceName(ctx, "enrichment-service")
	a.Logger.InfowCtx(shutdownCtx, "Shutting down enrichment service")

	additionalShutdown := func(ctx context.Context) []error {
		var errs []error

		if a.server != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), constants.ShutdownTimeout)
			defer cancel()
			if err := a.server.Shutdown(shutdownCtx); err != nil {
				errs = append(errs, fmt.Errorf("HTTP server shutdown error: %w", err))
			}
		}

		errs = append(errs, a.dbConnector.ShutdownDatabases(ctx, a.redis, a.postgresDB, a.mongoClient)...)

		return errs
	}

	return a.Base.Shutdown(ctx, additionalShutdown)
}
