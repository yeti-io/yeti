package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yeti/internal/deduplication"
)

func TestDeduplicationRepository_SetNX(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)

	ctx := context.Background()
	repo := deduplication.NewRepository(infra.RedisClient)

	key := "test:dedup:key1"
	value := time.Now().Unix()
	ttl := 5 * time.Second

	success, err := repo.SetNX(ctx, key, value, ttl)
	require.NoError(t, err)
	assert.True(t, success)

	success, err = repo.SetNX(ctx, key, value+1, ttl)
	require.NoError(t, err)
	assert.False(t, success)
}

func TestDeduplicationRepository_SetNX_TTL(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := deduplication.NewRepository(infra.RedisClient)

	key := "test:dedup:key2"
	value := time.Now().Unix()
	ttl := 1 * time.Second

	success, err := repo.SetNX(ctx, key, value, ttl)
	require.NoError(t, err)
	assert.True(t, success)

	// Wait for TTL to expire
	time.Sleep(2 * time.Second)

	// Should be able to set again after TTL expires
	success, err = repo.SetNX(ctx, key, value+1, ttl)
	require.NoError(t, err)
	assert.True(t, success)
}

func TestDeduplicationRepository_SetNX_DifferentKeys(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := deduplication.NewRepository(infra.RedisClient)

	keys := []string{"test:dedup:key3", "test:dedup:key4", "test:dedup:key5"}
	ttl := 5 * time.Second

	// All keys should be set successfully
	for i, key := range keys {
		value := time.Now().Unix() + int64(i)
		success, err := repo.SetNX(ctx, key, value, ttl)
		require.NoError(t, err)
		assert.True(t, success, "key %s should be set successfully", key)
	}
}

func TestDeduplicationRepository_SetNX_ContextCancellation(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	repo := deduplication.NewRepository(infra.RedisClient)
	key := "test:dedup:key6"
	value := time.Now().Unix()
	ttl := 5 * time.Second

	// Should return error due to cancelled context
	_, err := repo.SetNX(ctx, key, value, ttl)
	require.Error(t, err)
}
