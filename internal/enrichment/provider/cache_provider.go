package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type CacheProvider struct {
	client *redis.Client
}

func NewCacheProvider(client *redis.Client) *CacheProvider {
	return &CacheProvider{
		client: client,
	}
}

func (p *CacheProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (map[string]interface{}, error) {
	if config.KeyPattern == "" {
		return nil, fmt.Errorf("key_pattern is required for cache provider")
	}

	key := config.KeyPattern
	key = strings.ReplaceAll(key, "{field_value}", fmt.Sprintf("%v", fieldValue))
	key = strings.ReplaceAll(key, "{value}", fmt.Sprintf("%v", fieldValue))

	val, err := p.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache key not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return map[string]interface{}{
			"value": val,
		}, nil
	}

	return result, nil
}
