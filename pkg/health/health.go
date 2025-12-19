package health

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

type Checker interface {
	Check(ctx context.Context) error
	Name() string
}

type Health struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

type CheckResult struct {
	Status    Status    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type CheckerRegistry struct {
	checkers []Checker
}

func NewCheckerRegistry() *CheckerRegistry {
	return &CheckerRegistry{
		checkers: make([]Checker, 0),
	}
}

func (r *CheckerRegistry) Register(checker Checker) {
	r.checkers = append(r.checkers, checker)
}

func (r *CheckerRegistry) Check(ctx context.Context) Health {
	results := make(map[string]CheckResult)
	allHealthy := true
	anyDegraded := false

	for _, checker := range r.checkers {
		err := checker.Check(ctx)
		result := CheckResult{
			Timestamp: time.Now(),
		}

		if err != nil {
			result.Status = StatusUnhealthy
			result.Message = err.Error()
			allHealthy = false
		} else {
			result.Status = StatusHealthy
		}

		results[checker.Name()] = result
	}

	overallStatus := StatusHealthy
	if !allHealthy {
		overallStatus = StatusUnhealthy
	} else if anyDegraded {
		overallStatus = StatusDegraded
	}

	return Health{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    results,
	}
}

type PostgreSQLChecker struct {
	db *sql.DB
}

func NewPostgreSQLChecker(db *sql.DB) *PostgreSQLChecker {
	return &PostgreSQLChecker{db: db}
}

func (c *PostgreSQLChecker) Name() string {
	return "postgresql"
}

func (c *PostgreSQLChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgresql ping failed: %w", err)
	}
	return nil
}

type RedisChecker struct {
	client *redis.Client
}

func NewRedisChecker(client *redis.Client) *RedisChecker {
	return &RedisChecker{client: client}
}

func (c *RedisChecker) Name() string {
	return "redis"
}

func (c *RedisChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

type MongoDBChecker struct {
	client *mongo.Client
}

func NewMongoDBChecker(client *mongo.Client) *MongoDBChecker {
	return &MongoDBChecker{client: client}
}

func (c *MongoDBChecker) Name() string {
	return "mongodb"
}

func (c *MongoDBChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongodb ping failed: %w", err)
	}
	return nil
}
