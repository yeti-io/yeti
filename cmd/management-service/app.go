package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"

	_ "github.com/lib/pq" // PostgreSQL driver

	"yeti/internal/broker"
	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/logger"
	"yeti/internal/management"
	"yeti/pkg/bootstrap"
	"yeti/pkg/health"
	"yeti/pkg/metrics"
	"yeti/pkg/middleware"
	"yeti/pkg/ratelimit"
	"yeti/pkg/tracing"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type App struct {
	config         *config.Config
	logger         logger.Logger
	dbConnector    *bootstrap.DatabaseConnector
	db             *sql.DB
	mongoClient    *mongo.Client
	server         *http.Server
	router         *gin.Engine
	tracerProvider *tracing.TracerProvider
}

func NewApp(cfg *config.Config, log logger.Logger) *App {
	return &App{
		config:      cfg,
		logger:      log,
		dbConnector: bootstrap.NewDatabaseConnector(cfg, log),
	}
}

func (a *App) Initialize(ctx context.Context) error {
	if err := a.initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := a.initRouter(); err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}

	if err := a.initServer(); err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	tp, err := tracing.Init(a.config.Tracing, "management-service")
	if err != nil {
		return fmt.Errorf("failed to initialize tracing: %w", err)
	}
	a.tracerProvider = tp

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

func (a *App) initMongoDB(ctx context.Context) error {
	if a.config.Database.MongoDB.URI == "" {
		return nil // MongoDB is optional
	}

	// MongoDB initialization will be done in initRouter where it's needed
	// This is a placeholder for now
	return nil
}

func (a *App) initRouter() error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	if a.config.Tracing.Enabled {
		router.Use(tracing.GinMiddleware("management-service"))
	}

	router.Use(middleware.RecoveryMiddleware(a.logger))
	router.Use(middleware.LoggerMiddleware(a.logger))
	router.Use(middleware.RequestIDMiddleware())

	if a.config.Management.RateLimit.Enabled {
		rateLimitConfig := ratelimit.RateLimitConfig{
			RPS:             a.config.Management.RateLimit.RPS,
			Burst:           a.config.Management.RateLimit.Burst,
			CleanupInterval: time.Duration(a.config.Management.RateLimit.CleanupInterval) * time.Second,
			MaxAge:          time.Duration(a.config.Management.RateLimit.MaxAge) * time.Second,
		}
		router.Use(ratelimit.RateLimitMiddleware(rateLimitConfig))
		a.logger.InfowCtx(context.Background(), "Rate limiting enabled", "rps", rateLimitConfig.RPS, "burst", rateLimitConfig.Burst)
	}

	repo := management.NewRepository(a.db)
	versioningRepo := management.NewVersioningRepository(a.db)

	var enrichmentRepo management.EnrichmentRepository
	if a.config.Database.MongoDB.URI != "" {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		mongoClient, err := a.dbConnector.InitMongoDB(initCtx)
		if err != nil {
			a.logger.WarnwCtx(initCtx, "MongoDB connection failed, continuing without MongoDB", "error", err)
		} else if mongoClient != nil {
			a.mongoClient = mongoClient
			dbName := a.config.Database.MongoDB.Database
			if dbName == "" {
				dbName = constants.DefaultMongoDBName
			}
			mongoDB := mongoClient.Database(dbName)

			enrichmentRepo = management.NewEnrichmentRepository(mongoDB)
		}
	}

	var configEventProducer *management.ConfigEventProducer
	if a.config.Broker.Type == "kafka" && a.config.Broker.Kafka.ConfigUpdateTopic != "" {
		producer, err := broker.NewProducer(a.config.Broker, a.logger)
		if err != nil {
			a.logger.WarnwCtx(context.Background(), "Failed to create config event producer, config events will be disabled", "error", err)
		} else {
			configEventProducer = management.NewConfigEventProducer(producer, a.config.Broker.Kafka.ConfigUpdateTopic)
			a.logger.InfowCtx(context.Background(), "Config event producer initialized")
		}
	}

	opts := []management.ServiceOption{}
	if versioningRepo != nil {
		opts = append(opts, management.WithVersioning(versioningRepo))
	}
	if enrichmentRepo != nil {
		opts = append(opts, management.WithEnrichment(enrichmentRepo))
	}
	if configEventProducer != nil {
		opts = append(opts, management.WithConfigEvents(configEventProducer))
	}
	if a.config.Deduplication.HashAlgorithm != "" {
		opts = append(opts, management.WithDeduplicationConfig(a.config.Deduplication))
	}

	svc := management.NewService(repo, opts...)

	filteringHandler := management.NewHandler(svc, a.logger)
	enrichmentHandler := management.NewEnrichmentHandler(svc, a.logger)
	deduplicationHandler := management.NewDeduplicationHandler(svc, a.logger)

	filteringHandler.RegisterRoutes(router)
	if enrichmentHandler != nil {
		enrichmentHandler.RegisterEnrichmentRoutes(router)
	}
	if deduplicationHandler != nil {
		deduplicationHandler.RegisterDeduplicationRoutes(router)
	}

	metrics.RegisterManagementMetrics()
	metrics.RegisterCircuitBreakerMetrics()

	healthRegistry := health.NewCheckerRegistry()
	healthRegistry.Register(health.NewPostgreSQLChecker(a.db))
	if a.mongoClient != nil {
		healthRegistry.Register(health.NewMongoDBChecker(a.mongoClient))
	}

	router.GET("/health", func(c *gin.Context) {
		h := healthRegistry.Check(c.Request.Context())
		statusCode := http.StatusOK
		if h.Status == health.StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if h.Status == health.StatusDegraded {
			statusCode = http.StatusOK
		}
		c.JSON(statusCode, h)
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	a.router = router
	return nil
}

func (a *App) initServer() error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.config.Server.Port),
		Handler: a.router,
	}
	return nil
}

func (a *App) Run(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		a.logger.InfowCtx(ctx, "Server listening", "port", a.config.Server.Port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		return a.Shutdown(ctx)
	case err := <-errChan:
		return err
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.InfowCtx(ctx, "Shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), constants.ShutdownTimeout)
	defer cancel()

	var errs []error

	if a.server != nil {
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("server shutdown error: %w", err))
		}
	}

	if a.tracerProvider != nil {
		if err := a.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer provider shutdown error: %w", err))
		}
	}

	dbErrs := a.dbConnector.ShutdownDatabases(ctx, nil, a.db, a.mongoClient)
	errs = append(errs, dbErrs...)

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	a.logger.InfowCtx(ctx, "Server exited successfully")
	return nil
}
