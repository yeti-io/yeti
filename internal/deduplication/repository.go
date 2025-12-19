package deduplication

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Repository interface {
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
	GetCacheSize(ctx context.Context, prefix string) (int, error)
}

type RedisRepository struct {
	client *redis.Client
}

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
