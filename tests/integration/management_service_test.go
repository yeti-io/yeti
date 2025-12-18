package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yeti/internal/config"
	"yeti/internal/constants"
	"yeti/internal/management"
	pkgerrors "yeti/pkg/errors"
)

func TestManagementService_CreateFilteringRule(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
		Enabled:    boolPtr(true),
	}

	rule, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID)
	assert.Equal(t, req.Name, rule.Name)
	assert.Equal(t, req.Expression, rule.Expression)
	assert.Equal(t, req.Priority, rule.Priority)
	assert.True(t, rule.Enabled)
}

func TestManagementService_CreateFilteringRule_ValidationError_EmptyName(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	rule, err := svc.CreateFilteringRule(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.True(t, pkgerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "name is required")
}

func TestManagementService_CreateFilteringRule_ValidationError_EmptyExpression(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "",
		Priority:   10,
	}

	rule, err := svc.CreateFilteringRule(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.True(t, pkgerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "expression is required")
}

func TestManagementService_CreateFilteringRule_ValidationError_InvalidCEL(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "invalid syntax!!!",
		Priority:   10,
	}

	rule, err := svc.CreateFilteringRule(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.True(t, pkgerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid CEL expression")
}

func TestManagementService_CreateFilteringRule_ValidationError_NonBoolCEL(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status",
		Priority:   10,
	}

	rule, err := svc.CreateFilteringRule(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.True(t, pkgerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid CEL expression")
}

func TestManagementService_GetFilteringRule(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	created, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)

	retrieved, err := svc.GetFilteringRule(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Name, retrieved.Name)
	assert.Equal(t, created.Expression, retrieved.Expression)
}

func TestManagementService_GetFilteringRule_NotFound(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	rule, err := svc.GetFilteringRule(ctx, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.True(t, pkgerrors.IsNotFound(err))
}

func TestManagementService_ListFilteringRules(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req1 := management.CreateFilteringRuleRequest{
		Name:       "rule1",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}
	req2 := management.CreateFilteringRuleRequest{
		Name:       "rule2",
		Expression: "payload.type == 'event'",
		Priority:   20,
	}

	_, err := svc.CreateFilteringRule(ctx, req1)
	require.NoError(t, err)
	time.Sleep(timestampDelay)
	_, err = svc.CreateFilteringRule(ctx, req2)
	require.NoError(t, err)

	rules, err := svc.ListFilteringRules(ctx)
	require.NoError(t, err)
	assert.Len(t, rules, 2)
}

func TestManagementService_UpdateFilteringRule(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
		Enabled:    boolPtr(true),
	}

	created, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)

	updateReq := management.UpdateFilteringRuleRequest{
		Name:       stringPtr("updated_rule"),
		Expression: stringPtr("payload.status == 'inactive'"),
		Priority:   intPtr(15),
		Enabled:    boolPtr(false),
	}

	updated, err := svc.UpdateFilteringRule(ctx, created.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "updated_rule", updated.Name)
	assert.Equal(t, "payload.status == 'inactive'", updated.Expression)
	assert.Equal(t, 15, updated.Priority)
	assert.False(t, updated.Enabled)
}

func TestManagementService_UpdateFilteringRule_ValidationError_InvalidCEL(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	created, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)

	updateReq := management.UpdateFilteringRuleRequest{
		Expression: stringPtr("invalid syntax!!!"),
	}

	updated, err := svc.UpdateFilteringRule(ctx, created.ID, updateReq)
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.True(t, pkgerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid CEL expression")
}

func TestManagementService_UpdateFilteringRule_NotFound(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	updateReq := management.UpdateFilteringRuleRequest{
		Name: stringPtr("updated_rule"),
	}

	updated, err := svc.UpdateFilteringRule(ctx, "00000000-0000-0000-0000-000000000000", updateReq)
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.True(t, pkgerrors.IsNotFound(err))
}

func TestManagementService_DeleteFilteringRule(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	created, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)

	err = svc.DeleteFilteringRule(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetFilteringRule(ctx, created.ID)
	assert.Error(t, err)
	assert.True(t, pkgerrors.IsNotFound(err))
}

func TestManagementService_DeleteFilteringRule_NotFound(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	err := svc.DeleteFilteringRule(ctx, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
	assert.True(t, pkgerrors.IsNotFound(err))
}

func TestManagementService_CreateFilteringRule_WithVersioning(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	versioningRepo := management.NewVersioningRepository(infra.PostgresDB)
	svc := management.NewService(repo, management.WithVersioning(versioningRepo))

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	rule, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)

	versions, err := svc.GetRuleVersions(ctx, rule.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 1)
	assert.Equal(t, 1, versions[0].Version)
	assert.Equal(t, rule.ID, versions[0].RuleID)
	assert.Equal(t, "filtering", versions[0].RuleType)
}

func TestManagementService_UpdateFilteringRule_WithVersioning(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	versioningRepo := management.NewVersioningRepository(infra.PostgresDB)
	svc := management.NewService(repo, management.WithVersioning(versioningRepo))

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	created, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)

	updateReq := management.UpdateFilteringRuleRequest{
		Name: stringPtr("updated_rule"),
	}

	_, err = svc.UpdateFilteringRule(ctx, created.ID, updateReq)
	require.NoError(t, err)

	versions, err := svc.GetRuleVersions(ctx, created.ID)
	require.NoError(t, err)
	assert.Len(t, versions, 2)
	assert.Equal(t, 2, versions[0].Version)
	assert.Equal(t, 1, versions[1].Version)
}

func TestManagementService_GetAuditLogs(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	versioningRepo := management.NewVersioningRepository(infra.PostgresDB)
	svc := management.NewService(repo, management.WithVersioning(versioningRepo))

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	created, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)

	updateReq := management.UpdateFilteringRuleRequest{
		Name: stringPtr("updated_rule"),
	}

	_, err = svc.UpdateFilteringRule(ctx, created.ID, updateReq)
	require.NoError(t, err)

	logs, err := svc.GetAuditLogs(ctx, &created.ID, "filtering", 100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logs), 1, "Should have at least one audit log")

	hasCreate := false
	hasUpdate := false
	for _, log := range logs {
		if log.Action == "create" {
			hasCreate = true
		}
		if log.Action == "update" {
			hasUpdate = true
		}
	}
	assert.True(t, hasCreate || hasUpdate, "Should have create or update action")
	assert.True(t, hasUpdate, "Should have update action")
}

func TestManagementService_GetAuditLogs_WithoutVersioning(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	logs, err := svc.GetAuditLogs(ctx, nil, "filtering", 100)
	assert.Error(t, err)
	assert.Nil(t, logs)
	assert.Contains(t, err.Error(), "audit logging not enabled")
}

func TestManagementService_GetRuleVersions_WithoutVersioning(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	created, err := svc.CreateFilteringRule(ctx, req)
	require.NoError(t, err)

	versions, err := svc.GetRuleVersions(ctx, created.ID)
	assert.Error(t, err)
	assert.Nil(t, versions)
	assert.Contains(t, err.Error(), "versioning not enabled")
}

func TestManagementService_CreateEnrichmentRule(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	enrichmentRepo := management.NewEnrichmentRepository(infra.MongoDB)
	svc := management.NewService(repo, management.WithEnrichment(enrichmentRepo))

	req := management.CreateEnrichmentRuleRequest{
		Name:          "test_enrichment_rule",
		FieldToEnrich: "user_id",
		SourceType:    "cache",
		SourceConfig: management.EnrichmentSourceConfig{
			KeyPattern: "user:{user_id}",
		},
		Transformations: []management.EnrichmentTransformation{
			{
				SourcePath:  "name",
				TargetField: "user_name",
			},
		},
		Priority: 10,
		Enabled:  boolPtr(true),
	}

	rule, err := svc.CreateEnrichmentRule(ctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID)
	assert.Equal(t, req.Name, rule.Name)
	assert.Equal(t, req.FieldToEnrich, rule.FieldToEnrich)
	assert.Equal(t, req.SourceType, rule.SourceType)
}

func TestManagementService_CreateEnrichmentRule_WithoutEnrichmentRepo(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	req := management.CreateEnrichmentRuleRequest{
		Name:          "test_enrichment_rule",
		FieldToEnrich: "user_id",
		SourceType:    "cache",
		SourceConfig: management.EnrichmentSourceConfig{
			KeyPattern: "user:{user_id}",
		},
	}

	rule, err := svc.CreateEnrichmentRule(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Contains(t, err.Error(), "enrichment repository not configured")
}

func TestManagementService_CreateEnrichmentRule_ValidationError(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	enrichmentRepo := management.NewEnrichmentRepository(infra.MongoDB)
	svc := management.NewService(repo, management.WithEnrichment(enrichmentRepo))

	req := management.CreateEnrichmentRuleRequest{
		Name:          "",
		FieldToEnrich: "user_id",
		SourceType:    "cache",
		SourceConfig: management.EnrichmentSourceConfig{
			KeyPattern: "user:{user_id}",
		},
	}

	rule, err := svc.CreateEnrichmentRule(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.True(t, pkgerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "name is required")
}

func TestManagementService_GetEnrichmentRule(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	enrichmentRepo := management.NewEnrichmentRepository(infra.MongoDB)
	svc := management.NewService(repo, management.WithEnrichment(enrichmentRepo))

	req := management.CreateEnrichmentRuleRequest{
		Name:          "test_enrichment_rule",
		FieldToEnrich: "user_id",
		SourceType:    "cache",
		SourceConfig: management.EnrichmentSourceConfig{
			KeyPattern: "user:{user_id}",
		},
	}

	created, err := svc.CreateEnrichmentRule(ctx, req)
	require.NoError(t, err)

	retrieved, err := svc.GetEnrichmentRule(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, retrieved.ID)
	assert.Equal(t, created.Name, retrieved.Name)
}

func TestManagementService_GetEnrichmentRule_NotFound(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	enrichmentRepo := management.NewEnrichmentRepository(infra.MongoDB)
	svc := management.NewService(repo, management.WithEnrichment(enrichmentRepo))

	rule, err := svc.GetEnrichmentRule(ctx, "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.True(t, pkgerrors.IsNotFound(err))
}

func TestManagementService_UpdateEnrichmentRule(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	enrichmentRepo := management.NewEnrichmentRepository(infra.MongoDB)
	svc := management.NewService(repo, management.WithEnrichment(enrichmentRepo))

	req := management.CreateEnrichmentRuleRequest{
		Name:          "test_enrichment_rule",
		FieldToEnrich: "user_id",
		SourceType:    "cache",
		SourceConfig: management.EnrichmentSourceConfig{
			KeyPattern: "user:{user_id}",
		},
	}

	created, err := svc.CreateEnrichmentRule(ctx, req)
	require.NoError(t, err)

	updateReq := management.UpdateEnrichmentRuleRequest{
		Name: stringPtr("updated_enrichment_rule"),
	}

	updated, err := svc.UpdateEnrichmentRule(ctx, created.ID, updateReq)
	require.NoError(t, err)
	assert.Equal(t, "updated_enrichment_rule", updated.Name)
}

func TestManagementService_DeleteEnrichmentRule(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	enrichmentRepo := management.NewEnrichmentRepository(infra.MongoDB)
	svc := management.NewService(repo, management.WithEnrichment(enrichmentRepo))

	req := management.CreateEnrichmentRuleRequest{
		Name:          "test_enrichment_rule",
		FieldToEnrich: "user_id",
		SourceType:    "cache",
		SourceConfig: management.EnrichmentSourceConfig{
			KeyPattern: "user:{user_id}",
		},
	}

	created, err := svc.CreateEnrichmentRule(ctx, req)
	require.NoError(t, err)

	err = svc.DeleteEnrichmentRule(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetEnrichmentRule(ctx, created.ID)
	assert.Error(t, err)
	assert.True(t, pkgerrors.IsNotFound(err))
}

func TestManagementService_GetDeduplicationConfig_NotInitialized(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	config, err := svc.GetDeduplicationConfig(ctx)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "deduplication config not initialized")
}

func TestManagementService_UpdateDeduplicationConfig(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	dedupCfg := config.DeduplicationConfig{
		HashAlgorithm: "md5",
		TTLSeconds:    300,
		OnRedisError:  constants.FallbackAllow,
		FieldsToHash:  []string{"id", "source"},
	}
	svc := management.NewService(repo, management.WithDeduplicationConfig(dedupCfg))

	req := management.UpdateDeduplicationConfigRequest{
		HashAlgorithm: stringPtr("sha256"),
		TTLSeconds:    intPtr(600),
	}

	updated, err := svc.UpdateDeduplicationConfig(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "sha256", updated.HashAlgorithm)
	assert.Equal(t, 600, updated.TTLSeconds)
	assert.Equal(t, constants.FallbackAllow, updated.OnRedisError)
}

func TestManagementService_GetDeduplicationConfig(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	dedupCfg := config.DeduplicationConfig{
		HashAlgorithm: "md5",
		TTLSeconds:    300,
		OnRedisError:  constants.FallbackAllow,
		FieldsToHash:  []string{"id", "source"},
	}
	svc := management.NewService(repo, management.WithDeduplicationConfig(dedupCfg))

	config, err := svc.GetDeduplicationConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "md5", config.HashAlgorithm)
	assert.Equal(t, 300, config.TTLSeconds)
	assert.Equal(t, constants.FallbackAllow, config.OnRedisError)
	assert.Equal(t, []string{"id", "source"}, config.FieldsToHash)
}

func TestManagementService_UpdateDeduplicationConfig_ValidationError(t *testing.T) {
	infra := SetupTestInfra(t)

	ctx := context.Background()
	repo := management.NewRepository(infra.PostgresDB)
	dedupCfg := config.DeduplicationConfig{
		HashAlgorithm: "md5",
		TTLSeconds:    300,
		OnRedisError:  constants.FallbackAllow,
		FieldsToHash:  []string{"id", "source"},
	}
	svc := management.NewService(repo, management.WithDeduplicationConfig(dedupCfg))

	req := management.UpdateDeduplicationConfigRequest{
		HashAlgorithm: stringPtr("invalid_algorithm"),
	}

	updated, err := svc.UpdateDeduplicationConfig(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, updated)
	assert.True(t, pkgerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "invalid hash_algorithm")
}

func TestManagementService_CreateFilteringRule_ContextTimeout(t *testing.T) {
	infra := SetupTestInfra(t)

	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	rule, err := svc.CreateFilteringRule(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestManagementService_CreateFilteringRule_ContextCancellation(t *testing.T) {
	infra := SetupTestInfra(t)

	repo := management.NewRepository(infra.PostgresDB)
	svc := management.NewService(repo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
	}

	rule, err := svc.CreateFilteringRule(ctx, req)
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Contains(t, err.Error(), "context canceled")
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
