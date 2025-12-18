package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yeti/internal/deduplication"
)

func TestDeduplicationService_Process_Unique(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	ctx := context.Background()
	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	svc := deduplication.NewService(repo, cfg, log)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"data": "value"})

	isUnique, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.True(t, isUnique)
}

func TestDeduplicationService_Process_Duplicate(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	ctx := context.Background()
	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	svc := deduplication.NewService(repo, cfg, log)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"data": "value"})

	isUnique, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.True(t, isUnique)

	isUnique, err = svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.False(t, isUnique)
}

func TestDeduplicationService_Process_DifferentMessages(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	ctx := context.Background()
	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	svc := deduplication.NewService(repo, cfg, log)

	msg1 := createTestMessage("msg-1", "test", map[string]interface{}{"data": "value1"})
	msg2 := createTestMessage("msg-2", "test", map[string]interface{}{"data": "value2"})

	isUnique, err := svc.Process(ctx, msg1)
	require.NoError(t, err)
	assert.True(t, isUnique)

	isUnique, err = svc.Process(ctx, msg2)
	require.NoError(t, err)
	assert.True(t, isUnique)
}

func TestDeduplicationService_Process_CustomFields(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	ctx := context.Background()
	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfigWithFields([]string{"payload.user_id", "payload.order_id"})
	svc := deduplication.NewService(repo, cfg, log)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id":  "user-123",
		"order_id": "order-456",
	})

	isUnique, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.True(t, isUnique)

	isUnique, err = svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.False(t, isUnique)
}

func TestDeduplicationService_UpdateFieldsToHash(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	svc := deduplication.NewService(repo, cfg, log)

	err := svc.UpdateFieldsToHash([]string{"payload.field1", "payload.field2"})
	require.NoError(t, err)

	fields := svc.GetFieldsToHash()
	assert.Equal(t, []string{"payload.field1", "payload.field2"}, fields)
}

// TestDeduplicationService_Process_FallbackAllow_OnRedisError tests that when Redis
// returns an error and fallback is set to "allow", the message is allowed
func TestDeduplicationService_Process_FallbackAllow_OnRedisError(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	ctx := context.Background()
	log := createTestLogger()

	// Close Redis connection to simulate error
	infra.RedisClient.Close()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	cfg.OnRedisError = "allow" // Set fallback to allow
	svc := deduplication.NewService(repo, cfg, log)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"data": "value"})

	// With fallback allow, even if Redis fails, message should be allowed
	isUnique, err := svc.Process(ctx, msg)
	require.NoError(t, err, "Should not return error when fallback is 'allow'")
	assert.True(t, isUnique, "Message should be allowed when Redis fails and fallback is 'allow'")
}

// TestDeduplicationService_Process_FallbackDeny_OnRedisError tests that when Redis
// returns an error and fallback is set to "deny", the message is denied
func TestDeduplicationService_Process_FallbackDeny_OnRedisError(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	ctx := context.Background()
	log := createTestLogger()

	// Close Redis connection to simulate error
	infra.RedisClient.Close()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	cfg.OnRedisError = "deny" // Set fallback to deny
	svc := deduplication.NewService(repo, cfg, log)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"data": "value"})

	// With fallback deny, if Redis fails, message should be denied with error
	isUnique, err := svc.Process(ctx, msg)
	assert.Error(t, err, "Should return error when fallback is 'deny'")
	assert.False(t, isUnique, "Message should be denied when Redis fails and fallback is 'deny'")
	assert.Contains(t, err.Error(), "redis error")
}

// TestDeduplicationService_Process_SHA256Hash tests that SHA256 hash algorithm works correctly
func TestDeduplicationService_Process_SHA256Hash(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	ctx := context.Background()
	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	cfg.HashAlgorithm = "sha256" // Use SHA256 instead of default MD5
	svc := deduplication.NewService(repo, cfg, log)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"data": "value"})

	// First message should be unique
	isUnique, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.True(t, isUnique)

	// Same message should be duplicate
	isUnique, err = svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.False(t, isUnique)

	// Different message should be unique
	msg2 := createTestMessage("msg-2", "test", map[string]interface{}{"data": "different"})
	isUnique, err = svc.Process(ctx, msg2)
	require.NoError(t, err)
	assert.True(t, isUnique)
}

// TestDeduplicationService_UpdateFieldsToHash_EmptyList tests that error occurs when
// trying to update fields to hash with an empty list
func TestDeduplicationService_UpdateFieldsToHash_EmptyList(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	svc := deduplication.NewService(repo, cfg, log)

	// Trying to set empty fields list should return an error
	err := svc.UpdateFieldsToHash([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fields list cannot be empty")
}

// TestDeduplicationService_Process_ContextTimeout tests that deduplication respects context timeout
func TestDeduplicationService_Process_ContextTimeout(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	svc := deduplication.NewService(repo, cfg, log)

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	// Wait a bit to ensure timeout
	time.Sleep(10 * time.Millisecond)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"data": "value"})

	// Should return context deadline exceeded error
	isUnique, err := svc.Process(ctx, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.False(t, isUnique)
}

// TestDeduplicationService_Process_ContextCancellation tests that deduplication respects context cancellation
func TestDeduplicationService_Process_ContextCancellation(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, false, true)
	

	log := createTestLogger()

	repo := deduplication.NewRepository(infra.RedisClient)
	cfg := createTestDeduplicationConfig()
	svc := deduplication.NewService(repo, cfg, log)

	// Create a context and cancel it immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"data": "value"})

	// Should return context canceled error
	isUnique, err := svc.Process(ctx, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.False(t, isUnique)
}
