package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"yeti/internal/enrichment"
)

func TestEnrichmentRepository_GetActiveRules(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, false)
	

	ctx := context.Background()

	// Set up test data directly in MongoDB
	collection := infra.MongoDB.Collection("enrichment_rules")

	rules := []enrichment.Rule{
		{
			Name:            "active_rule_1",
			FieldToEnrich:   "user_id",
			SourceType:      "api",
			SourceConfig:    enrichment.SourceConfig{URL: "http://api.example.com/user"},
			Transformations: []enrichment.Transformation{},
			CacheTTLSeconds: 300,
			ErrorHandling:   "fail",
			Priority:        10,
			Enabled:         true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			Name:            "active_rule_2",
			FieldToEnrich:   "product_id",
			SourceType:      "database",
			SourceConfig:    enrichment.SourceConfig{Database: "products", Collection: "items"},
			Transformations: []enrichment.Transformation{},
			CacheTTLSeconds: 600,
			ErrorHandling:   "skip_rule",
			Priority:        20,
			Enabled:         true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			Name:            "inactive_rule",
			FieldToEnrich:   "order_id",
			SourceType:      "cache",
			SourceConfig:    enrichment.SourceConfig{KeyPattern: "order:{value}"},
			Transformations: []enrichment.Transformation{},
			CacheTTLSeconds: 60,
			ErrorHandling:   "skip_field",
			Priority:        5,
			Enabled:         false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	for _, rule := range rules {
		_, err := collection.InsertOne(ctx, rule)
		require.NoError(t, err)
	}

	// Test repository
	repo := enrichment.NewRepository(infra.MongoDB)
	activeRules, err := repo.GetActiveRules(ctx)
	require.NoError(t, err)

	// Should only return enabled rules, ordered by priority ASC
	assert.Len(t, activeRules, 2)
	assert.Equal(t, "active_rule_1", activeRules[0].Name) // Priority 10
	assert.Equal(t, "active_rule_2", activeRules[1].Name) // Priority 20
}

func TestEnrichmentRepository_GetActiveRules_Empty(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, false)
	

	ctx := context.Background()

	repo := enrichment.NewRepository(infra.MongoDB)
	activeRules, err := repo.GetActiveRules(ctx)
	require.NoError(t, err)
	assert.Empty(t, activeRules)
}

func TestEnrichmentRepository_GetActiveRules_Ordering(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, false)
	

	ctx := context.Background()

	collection := infra.MongoDB.Collection("enrichment_rules")

	// Create rules with different priorities
	rules := []enrichment.Rule{
		{Name: "low_priority", FieldToEnrich: "field1", SourceType: "api", Priority: 5, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Name: "high_priority", FieldToEnrich: "field2", SourceType: "api", Priority: 50, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Name: "medium_priority", FieldToEnrich: "field3", SourceType: "api", Priority: 25, Enabled: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, rule := range rules {
		_, err := collection.InsertOne(ctx, rule)
		require.NoError(t, err)
	}

	repo := enrichment.NewRepository(infra.MongoDB)
	activeRules, err := repo.GetActiveRules(ctx)
	require.NoError(t, err)

	// Should be ordered by priority ASC
	assert.Len(t, activeRules, 3)
	assert.Equal(t, "low_priority", activeRules[0].Name)   // Priority 5
	assert.Equal(t, "medium_priority", activeRules[1].Name) // Priority 25
	assert.Equal(t, "high_priority", activeRules[2].Name)   // Priority 50
}

func TestEnrichmentRepository_GetActiveRules_Filter(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, false)
	

	ctx := context.Background()

	collection := infra.MongoDB.Collection("enrichment_rules")

	// Insert test data
	_, err := collection.InsertMany(ctx, []interface{}{
		bson.M{"name": "enabled_rule", "enabled": true, "priority": 10, "field_to_enrich": "field1", "source_type": "api", "created_at": time.Now(), "updated_at": time.Now()},
		bson.M{"name": "disabled_rule", "enabled": false, "priority": 10, "field_to_enrich": "field2", "source_type": "api", "created_at": time.Now(), "updated_at": time.Now()},
	})
	require.NoError(t, err)

	repo := enrichment.NewRepository(infra.MongoDB)
	activeRules, err := repo.GetActiveRules(ctx)
	require.NoError(t, err)

	// Should only return enabled rules
	assert.Len(t, activeRules, 1)
	assert.Equal(t, "enabled_rule", activeRules[0].Name)
	assert.True(t, activeRules[0].Enabled)
}
