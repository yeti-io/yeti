package bootstrap

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yeti/internal/config"
	"yeti/internal/logger"
)

type DatabaseConnector struct {
	Config *config.Config
	Logger logger.Logger
}

func NewDatabaseConnector(cfg *config.Config, log logger.Logger) *DatabaseConnector {
	return &DatabaseConnector{
		Config: cfg,
		Logger: log,
	}
}

func (dc *DatabaseConnector) InitRedis(ctx context.Context) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", dc.Config.Database.Redis.Host, dc.Config.Database.Redis.Port),
		Password: dc.Config.Database.Redis.Password,
		DB:       dc.Config.Database.Redis.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	dc.Logger.Info("Redis connected successfully")
	return rdb, nil
}

func (dc *DatabaseConnector) InitPostgreSQL(ctx context.Context) (*sql.DB, error) {
	if dc.Config.Database.Postgres.Host == "" {
		return nil, nil // PostgreSQL is optional
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dc.Config.Database.Postgres.User,
		dc.Config.Database.Postgres.Password,
		dc.Config.Database.Postgres.Host,
		dc.Config.Database.Postgres.Port,
		dc.Config.Database.Postgres.DBName,
		dc.Config.Database.Postgres.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	dc.Logger.Info("PostgreSQL connected successfully")
	return db, nil
}

func (dc *DatabaseConnector) InitMongoDB(ctx context.Context) (*mongo.Client, error) {
	if dc.Config.Database.MongoDB.URI == "" {
		return nil, nil // MongoDB is optional
	}

	mongoOpts := options.Client().ApplyURI(dc.Config.Database.MongoDB.URI)
	mongoClient, err := mongo.Connect(ctx, mongoOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		mongoClient.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	dc.Logger.Info("MongoDB connected successfully")
	return mongoClient, nil
}

func (dc *DatabaseConnector) ShutdownDatabases(ctx context.Context, redis *redis.Client, postgres *sql.DB, mongo *mongo.Client) []error {
	var errs []error

	if redis != nil {
		if err := redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("redis close error: %w", err))
		}
	}

	if postgres != nil {
		if err := postgres.Close(); err != nil {
			errs = append(errs, fmt.Errorf("postgres close error: %w", err))
		}
	}

	if mongo != nil {
		if err := mongo.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("mongodb disconnect error: %w", err))
		}
	}

	return errs
}
