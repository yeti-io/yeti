package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	postgresmodule "github.com/testcontainers/testcontainers-go/modules/postgres"
	redismodule "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type TestInfra struct {
	PostgresDB   *sql.DB
	PostgresConn string
	MongoDB      *mongo.Database
	MongoClient  *mongo.Client
	RedisClient  *redisclient.Client
}

func SetupTestInfra(t *testing.T) *TestInfra {
	return SetupTestInfraWithOptions(t, true, true, true)
}

func SetupTestInfraWithOptions(t *testing.T, needPostgres, needMongo, needRedis bool) *TestInfra {
	t.Helper()

	ctx := context.Background()

	if os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "" {
		os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	}

	infra := &TestInfra{}

	if needPostgres {
		setupPostgres(t, ctx, infra)
	}

	if needMongo {
		setupMongo(t, ctx, infra)
	}

	if needRedis {
		setupRedis(t, ctx, infra)
	}

	return infra
}

func setupPostgres(t *testing.T, ctx context.Context, infra *TestInfra) {
	container, err := postgresmodule.Run(ctx, "postgres:15",
		postgresmodule.WithDatabase("test_db"),
		postgresmodule.WithUsername("test_user"),
		postgresmodule.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		container.Terminate(ctx)
	})

	conn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get postgres uri: %v", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", conn)
	if err != nil {
		t.Fatalf("failed to open postgres connection: %v", err)
	}

	if err := db.PingContext(ctxWithTimeout); err != nil {
		db.Close()
		t.Fatalf("failed to ping postgres: %v", err)
	}

	if err := runMigrations(db, conn); err != nil {
		db.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	infra.PostgresDB = db
	infra.PostgresConn = conn
	t.Cleanup(func() {
		db.Close()
	})
}

func setupMongo(t *testing.T, ctx context.Context, infra *TestInfra) {
	container, err := mongodb.Run(ctx, "mongo:6",
		mongodb.WithUsername("test_user"),
		mongodb.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Waiting for connections").WithStartupTimeout(containerStartupTimeout*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start mongo container: %v", err)
	}
	t.Cleanup(func() {
		container.Terminate(ctx)
	})

	port, err := container.MappedPort(ctx, "27017/tcp")
	if err != nil {
		t.Fatalf("failed to get mongo port: %v", err)
	}

	conn := fmt.Sprintf("mongodb://test_user:test_password@localhost:%s/test_db?authSource=admin", port.Port())

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(conn))
	if err != nil {
		t.Fatalf("failed to connect to mongo: %v", err)
	}

	infra.MongoDB = client.Database("test_db")
	infra.MongoClient = client
	t.Cleanup(func() {
		client.Disconnect(ctx)
	})
}

func setupRedis(t *testing.T, ctx context.Context, infra *TestInfra) {
	container, err := redismodule.Run(ctx, "redis:8.4.0-alpine")
	if err != nil {
		t.Fatalf("failed to start redis container: %v", err)
	}
	t.Cleanup(func() {
		container.Terminate(ctx)
	})

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get redis uri: %v", err)
	}

	opt, err := redisclient.ParseURL(uri)
	if err != nil {
		t.Fatalf("failed to parse redis URL: %v", err)
	}

	client := redisclient.NewClient(opt)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Ping(ctxWithTimeout).Err(); err != nil {
		client.Close()
		t.Fatalf("failed to ping redis: %v", err)
	}

	infra.RedisClient = client
	t.Cleanup(func() {
		client.Close()
	})
}

func runMigrations(db *sql.DB, connString string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get work dir: %w", err)
	}

	projectRoot := filepath.Join(workDir, "..", "..")
	migrationsPath := filepath.Join(projectRoot, "migrations", "postgres")

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
