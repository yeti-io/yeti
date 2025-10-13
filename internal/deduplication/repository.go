package deduplication

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Repository defines the interface for deduplication repository
type Repository interface {
	// SetNX sets a key if it doesn't exist (for deduplication check)
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
	// GetCacheSize returns the approximate number of keys with the given prefix
	GetCacheSize(ctx context.Context, prefix string) (int, error)
}

// RedisRepository implements the deduplication.Repository interface
type RedisRepository struct {
	client *redis.Client
}

// NewRepository creates a new Redis repository instance
func NewRepository(client *redis.Client) Repository {
	return &RedisRepository{client: client}
}

func (r *RedisRepository) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	success, err := r.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis SetNX failed: %w", err)
	}
	return success, nil
}

func (r *RedisRepository) GetCacheSize(ctx context.Context, prefix string) (int, error) {
	iter := r.client.Scan(ctx, 0, prefix+"*", 0).Iterator()
	count := 0
	for iter.Next(ctx) {
		// Check context cancellation to avoid infinite loop
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		count++
	}
	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("redis scan failed: %w", err)
	}
	return count, nil
}
