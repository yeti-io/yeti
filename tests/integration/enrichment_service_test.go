package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"yeti/internal/constants"
	"yeti/internal/enrichment"
)

func TestEnrichmentService_Process_CacheSource(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "cache_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: ".", TargetField: "user_data"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	userData := map[string]interface{}{"name": "John", "email": "john@example.com"}
	dataBytes, _ := json.Marshal(userData)
	infra.RedisClient.Set(ctx, "user:user-123", string(dataBytes), 300*time.Second)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, result.Metadata.Enrichment)
	assert.NotEmpty(t, result.Metadata.Enrichment["user_data"])
}

func TestEnrichmentService_Process_MongoDBSource(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	usersCollection := infra.MongoDB.Collection("users")
	_, err := usersCollection.InsertOne(ctx, bson.M{
		"user_id": "user-123",
		"name":    "John",
		"email":   "john@example.com",
	})
	require.NoError(t, err)

	rulesCollection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "mongodb_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeMongoDB,
		SourceConfig: enrichment.SourceConfig{
			Database:   "test_db",
			Collection: "users",
			Field:      "user_id",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: "name", TargetField: "user_name"},
			{SourcePath: "email", TargetField: "user_email"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = rulesCollection.InsertOne(ctx, rule)
	require.NoError(t, err)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewServiceWithDatabaseProviders(repo, infra.RedisClient, infra.MongoClient, nil, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, result.Metadata.Enrichment)
	assert.Equal(t, "John", result.Metadata.Enrichment["user_name"])
	assert.Equal(t, "john@example.com", result.Metadata.Enrichment["user_email"])
}

func TestEnrichmentService_Process_ErrorHandling_Fail(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "fail_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: ".", TargetField: "user_data"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingFail,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "non-existent",
		"data":    "value",
	})

	_, err = svc.Process(ctx, msg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enrichment failed")
}

func TestEnrichmentService_Process_ErrorHandling_SkipRule(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "skip_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: ".", TargetField: "user_data"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "non-existent",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.Empty(t, result.Metadata.Enrichment)
}

func TestEnrichmentService_Process_FallbackValue(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "fallback_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: ".", TargetField: "user_data"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		FallbackValue:  "default-user",
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "non-existent",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, result.Metadata.Enrichment)
	userData, ok := result.Metadata.Enrichment["user_data"].(map[string]interface{})
	require.True(t, ok, "user_data should be a map")
	assert.Equal(t, "default-user", userData["value"])
}

func TestEnrichmentService_Process_Transformations(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "transform_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: "name", TargetField: "user_name"},
			{SourcePath: "email", TargetField: "user_email"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	userData := map[string]interface{}{"name": "John", "email": "john@example.com"}
	dataBytes, _ := json.Marshal(userData)
	infra.RedisClient.Set(ctx, "user:user-123", string(dataBytes), 300*time.Second)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, result.Metadata.Enrichment)
	assert.Equal(t, "John", result.Metadata.Enrichment["user_name"])
	assert.Equal(t, "john@example.com", result.Metadata.Enrichment["user_email"])
}

func TestEnrichmentService_Process_NoRules(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err := svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.Empty(t, result.Metadata.Enrichment)
}

func TestEnrichmentService_Process_MissingField(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "missing_field_rule",
		FieldToEnrich: "non_existent_field",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: ".", TargetField: "user_data"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.Empty(t, result.Metadata.Enrichment)
}

func TestEnrichmentService_ReloadRules(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err := svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "new_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: ".", TargetField: "user_data"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err = collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	userData := map[string]interface{}{"name": "John"}
	dataBytes, _ := json.Marshal(userData)
	infra.RedisClient.Set(ctx, "user:user-123", string(dataBytes), 300*time.Second)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Metadata.Enrichment)
}

func TestEnrichmentService_Process_ErrorHandling_SkipField(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "skip_field_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: "name", TargetField: "user_name"},
			{SourcePath: "missing_field", TargetField: "missing_data"},
			{SourcePath: "email", TargetField: "user_email"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipField,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	userData := map[string]interface{}{"name": "John", "email": "john@example.com"}
	dataBytes, _ := json.Marshal(userData)
	infra.RedisClient.Set(ctx, "user:user-123", string(dataBytes), 300*time.Second)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, result.Metadata.Enrichment)
	assert.Equal(t, "John", result.Metadata.Enrichment["user_name"])
	assert.Equal(t, "john@example.com", result.Metadata.Enrichment["user_email"])
	assert.Nil(t, result.Metadata.Enrichment["missing_data"])
}

func TestEnrichmentService_Process_Transformations_Default(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "default_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: "name", TargetField: "user_name"},
			{SourcePath: "missing_field", TargetField: "missing_data", Default: "default-value"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	userData := map[string]interface{}{"name": "John"}
	dataBytes, _ := json.Marshal(userData)
	infra.RedisClient.Set(ctx, "user:user-123", string(dataBytes), 300*time.Second)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
		"data":    "value",
	})

	result, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, result.Metadata.Enrichment)
	assert.Equal(t, "John", result.Metadata.Enrichment["user_name"])
	assert.Equal(t, "default-value", result.Metadata.Enrichment["missing_data"])
}

func TestEnrichmentService_Process_CacheHit(t *testing.T) {
	infra := SetupTestInfraWithOptions(t, false, true, true)

	ctx := context.Background()
	log := createTestLogger()

	collection := infra.MongoDB.Collection("enrichment_rules")
	rule := enrichment.Rule{
		Name:          "cache_hit_rule",
		FieldToEnrich: "user_id",
		SourceType:    constants.SourceTypeCache,
		SourceConfig: enrichment.SourceConfig{
			KeyPattern: "user:{value}",
		},
		Transformations: []enrichment.Transformation{
			{SourcePath: "name", TargetField: "user_name"},
		},
		CacheTTLSeconds: 300,
		ErrorHandling:  constants.ErrorHandlingSkipRule,
		Priority:       10,
		Enabled:        true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	_, err := collection.InsertOne(ctx, rule)
	require.NoError(t, err)

	repo := enrichment.NewRepository(infra.MongoDB)
	svc := enrichment.NewService(repo, infra.RedisClient, log)

	err = svc.ReloadRules(ctx, true)
	require.NoError(t, err)

	userData := map[string]interface{}{"name": "John"}
	dataBytes, _ := json.Marshal(userData)
	infra.RedisClient.Set(ctx, "user:user-123", string(dataBytes), 300*time.Second)

	msg := createTestMessage("msg-1", "test", map[string]interface{}{
		"user_id": "user-123",
		"data":    "value",
	})

	processed, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, processed.Metadata.Enrichment)
	assert.Equal(t, "John", processed.Metadata.Enrichment["user_name"])

	processed2, err := svc.Process(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, processed2.Metadata.Enrichment)
	assert.Equal(t, "John", processed2.Metadata.Enrichment["user_name"])
}
