package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yeti/internal/filtering"
	"yeti/internal/management"
)


func TestFilteringService_Filter_Pass(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()
	log := createTestLogger()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)
	err := mgmtRepo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "active"})

	passed, appliedRules, err := svc.Filter(ctx, msg)
	require.NoError(t, err)
	assert.True(t, passed)
	assert.Len(t, appliedRules, 1)
	assert.Equal(t, rule.ID, appliedRules[0])
}

func TestFilteringService_Filter_Reject(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()
	log := createTestLogger()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)
	err := mgmtRepo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "inactive"})

	passed, appliedRules, err := svc.Filter(ctx, msg)
	require.NoError(t, err)
	assert.False(t, passed)
	assert.Empty(t, appliedRules)
}

func TestFilteringService_Filter_MultipleRules(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()
	log := createTestLogger()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	rules := []*management.FilteringRule{
		createTestFilteringRule("rule1", "payload.status == 'active'", 10, true),
		createTestFilteringRule("rule2", "payload.type == 'event'", 20, true),
	}

	for _, rule := range rules {
		err := mgmtRepo.CreateFilteringRule(ctx, rule)
		require.NoError(t, err)
		time.Sleep(timestampDelay)
	}

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "active", "type": "event"})

	passed, appliedRules, err := svc.Filter(ctx, msg)
	require.NoError(t, err)
	assert.True(t, passed)
	assert.Len(t, appliedRules, 2)
}

func TestFilteringService_ReloadRules(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()
	log := createTestLogger()

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)
	err = mgmtRepo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "active"})

	passed, appliedRules, err := svc.Filter(ctx, msg)
	require.NoError(t, err)
	assert.True(t, passed)
	assert.Len(t, appliedRules, 1)
}

// TestFilteringService_Filter_FallbackAllow_OnCELError tests that when a CEL expression
// fails to evaluate and fallback is set to "allow", the message is allowed
func TestFilteringService_Filter_FallbackAllow_OnCELError(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()
	log := createTestLogger()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	// Create a rule with an expression that will cause a runtime error
	// Using a non-existent field access that will fail at runtime
	rule := createTestFilteringRule("error_rule", "payload.nonexistent.field == 'value'", 10, true)
	err := mgmtRepo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	cfg.Fallback.OnError = "allow" // Set fallback to allow
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "active"})

	// With fallback allow, even if CEL evaluation fails, message should pass
	passed, appliedRules, err := svc.Filter(ctx, msg)
	require.NoError(t, err)
	assert.True(t, passed, "Message should be allowed when fallback is 'allow'")
	assert.Empty(t, appliedRules, "No rules should be applied when evaluation fails")
}

// TestFilteringService_Filter_FallbackDeny_OnCELError tests that when a CEL expression
// fails to evaluate and fallback is set to "deny", the message is denied
func TestFilteringService_Filter_FallbackDeny_OnCELError(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()
	log := createTestLogger()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	// Create a rule with an expression that will cause a runtime error
	rule := createTestFilteringRule("error_rule", "payload.nonexistent.field == 'value'", 10, true)
	err := mgmtRepo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	cfg.Fallback.OnError = "deny" // Set fallback to deny
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "active"})

	// With fallback deny, if CEL evaluation fails, message should be denied
	passed, appliedRules, err := svc.Filter(ctx, msg)
	require.NoError(t, err)
	assert.False(t, passed, "Message should be denied when fallback is 'deny'")
	assert.Empty(t, appliedRules, "No rules should be applied when evaluation fails")
}

// TestFilteringService_Filter_InvalidCELExpression tests that invalid CEL expressions
// are handled according to fallback strategy
func TestFilteringService_Filter_InvalidCELExpression(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()
	log := createTestLogger()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	// Create a rule with a syntactically invalid CEL expression
	// This will fail at compile time, not runtime
	// Note: We need to bypass validation to insert invalid expression
	// In real scenario, this shouldn't happen, but we test the behavior
	rule := createTestFilteringRule("invalid_rule", "invalid syntax here!!!", 10, true)
	err := mgmtRepo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	cfg.Fallback.OnError = "deny"
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "active"})

	// Invalid CEL expression should trigger fallback
	passed, appliedRules, err := svc.Filter(ctx, msg)
	require.NoError(t, err)
	assert.False(t, passed, "Message should be denied when CEL expression is invalid and fallback is 'deny'")
	assert.Empty(t, appliedRules)
}

// TestFilteringService_Filter_ContextTimeout tests that filtering respects context timeout
func TestFilteringService_Filter_ContextTimeout(t *testing.T) {
	infra := SetupTestInfra(t)
	

	log := createTestLogger()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)
	err := mgmtRepo.CreateFilteringRule(context.Background(), rule)
	require.NoError(t, err)

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(context.Background(), true)
	require.NoError(t, err)

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	// Wait a bit to ensure timeout
	time.Sleep(10 * time.Millisecond)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "active"})

	// Should return context deadline exceeded error
	passed, appliedRules, err := svc.Filter(ctx, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.False(t, passed)
	assert.Empty(t, appliedRules)
}

// TestFilteringService_Filter_ContextCancellation tests that filtering respects context cancellation
func TestFilteringService_Filter_ContextCancellation(t *testing.T) {
	infra := SetupTestInfra(t)
	

	log := createTestLogger()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)
	err := mgmtRepo.CreateFilteringRule(context.Background(), rule)
	require.NoError(t, err)

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	cfg := createTestFilteringConfig()
	svc, err := filtering.NewService(filteringRepo, cfg, log)
	require.NoError(t, err)

	err = svc.ReloadRules(context.Background(), true)
	require.NoError(t, err)

	// Create a context and cancel it immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	msg := createTestMessage("msg-1", "test", map[string]interface{}{"status": "active"})

	// Should return context canceled error
	passed, appliedRules, err := svc.Filter(ctx, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.False(t, passed)
	assert.Empty(t, appliedRules)
}
