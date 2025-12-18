package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yeti/internal/management"
)

func TestManagementRepository_CreateFilteringRule(t *testing.T) {
	infra := SetupTestInfra(t)

	repo := management.NewRepository(infra.PostgresDB)
	ctx := context.Background()

	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)

	err := repo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID)
	assert.False(t, rule.CreatedAt.IsZero())
	assert.False(t, rule.UpdatedAt.IsZero())
}

func TestManagementRepository_GetFilteringRule(t *testing.T) {
	infra := SetupTestInfra(t)

	repo := management.NewRepository(infra.PostgresDB)
	ctx := context.Background()

	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)
	err := repo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)

	retrieved, err := repo.GetFilteringRule(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, rule.ID, retrieved.ID)
	assert.Equal(t, rule.Name, retrieved.Name)
	assert.Equal(t, rule.Expression, retrieved.Expression)
	assert.Equal(t, rule.Priority, retrieved.Priority)
	assert.Equal(t, rule.Enabled, retrieved.Enabled)
}

func TestManagementRepository_GetFilteringRule_NotFound(t *testing.T) {
	infra := SetupTestInfra(t)

	repo := management.NewRepository(infra.PostgresDB)
	ctx := context.Background()

	_, err := repo.GetFilteringRule(ctx, "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManagementRepository_ListFilteringRules(t *testing.T) {
	infra := SetupTestInfra(t)

	repo := management.NewRepository(infra.PostgresDB)
	ctx := context.Background()

	rules := []*management.FilteringRule{
		createTestFilteringRule("rule1", "payload.status == 'active'", 10, true),
		createTestFilteringRule("rule2", "payload.type == 'event'", 20, true),
		createTestFilteringRule("rule3", "payload.value > 100", 5, false),
	}

	for _, rule := range rules {
		err := repo.CreateFilteringRule(ctx, rule)
		require.NoError(t, err)
		time.Sleep(timestampDelay)
	}

	list, err := repo.ListFilteringRules(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 3)

	assert.Equal(t, "rule2", list[0].Name) // Priority 20
	assert.Equal(t, "rule1", list[1].Name) // Priority 10
	assert.Equal(t, "rule3", list[2].Name) // Priority 5
}

func TestManagementRepository_UpdateFilteringRule(t *testing.T) {
	infra := SetupTestInfra(t)

	repo := management.NewRepository(infra.PostgresDB)
	ctx := context.Background()

	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)
	err := repo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)

	originalUpdatedAt := rule.UpdatedAt

	time.Sleep(timestampDelay)
	rule.Name = "updated_rule"
	rule.Expression = "payload.status == 'inactive'"
	rule.Priority = 15
	rule.Enabled = false

	err = repo.UpdateFilteringRule(ctx, rule)
	require.NoError(t, err)

	retrieved, err := repo.GetFilteringRule(ctx, rule.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated_rule", retrieved.Name)
	assert.Equal(t, "payload.status == 'inactive'", retrieved.Expression)
	assert.Equal(t, 15, retrieved.Priority)
	assert.False(t, retrieved.Enabled)
	assert.True(t, retrieved.UpdatedAt.After(originalUpdatedAt))
}

func TestManagementRepository_DeleteFilteringRule(t *testing.T) {
	infra := SetupTestInfra(t)

	repo := management.NewRepository(infra.PostgresDB)
	ctx := context.Background()

	rule := createTestFilteringRule("test_rule", "payload.status == 'active'", 10, true)
	err := repo.CreateFilteringRule(ctx, rule)
	require.NoError(t, err)
	err = repo.DeleteFilteringRule(ctx, rule.ID)
	require.NoError(t, err)

	_, err = repo.GetFilteringRule(ctx, rule.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
