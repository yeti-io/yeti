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

func TestFilteringRepository_GetActiveRules(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()

	mgmtRepo := management.NewRepository(infra.PostgresDB)
	rules := []*management.FilteringRule{
		createTestFilteringRule("active1", "payload.status == 'active'", 10, true),
		createTestFilteringRule("active2", "payload.type == 'event'", 20, true),
		createTestFilteringRule("inactive", "payload.value > 100", 5, false),
	}

	for _, rule := range rules {
		err := mgmtRepo.CreateFilteringRule(ctx, rule)
		require.NoError(t, err)
		time.Sleep(timestampDelay)
	}
	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	activeRules, err := filteringRepo.GetActiveRules(ctx)
	require.NoError(t, err)

	assert.Len(t, activeRules, 2)
	assert.Equal(t, "active2", activeRules[0].Name) // Priority 20
	assert.Equal(t, "active1", activeRules[1].Name) // Priority 10
}

func TestFilteringRepository_GetActiveRules_Empty(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	activeRules, err := filteringRepo.GetActiveRules(ctx)
	require.NoError(t, err)
	assert.Empty(t, activeRules)
}

func TestFilteringRepository_GetActiveRules_Ordering(t *testing.T) {
	infra := SetupTestInfra(t)
	

	ctx := context.Background()

	mgmtRepo := management.NewRepository(infra.PostgresDB)

	rules := []*management.FilteringRule{
		createTestFilteringRule("first", "payload.a == 1", 10, true),
		createTestFilteringRule("second", "payload.b == 2", 10, true),
		createTestFilteringRule("third", "payload.c == 3", 10, true),
	}

	for _, rule := range rules {
		err := mgmtRepo.CreateFilteringRule(ctx, rule)
		require.NoError(t, err)
		time.Sleep(timestampDelay)
	}

	filteringRepo := filtering.NewRepository(infra.PostgresDB)
	activeRules, err := filteringRepo.GetActiveRules(ctx)
	require.NoError(t, err)

	assert.Len(t, activeRules, 3)
	assert.Equal(t, "first", activeRules[0].Name)
	assert.Equal(t, "second", activeRules[1].Name)
	assert.Equal(t, "third", activeRules[2].Name)
}
