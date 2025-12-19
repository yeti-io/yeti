package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"yeti/internal/management"
)

const (
	managementServiceURL = "http://localhost:8084"
)

func TestManagementServiceHealth(t *testing.T) {
	url := fmt.Sprintf("%s/health", managementServiceURL)
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)
	assert.NotNil(t, health["status"])
}

func TestFilteringRulesCRUD(t *testing.T) {
	createReq := management.CreateFilteringRuleRequest{
		Name:       "test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
		Enabled:    boolPtr(true),
	}

	ruleID := createFilteringRule(t, createReq)
	defer deleteFilteringRule(t, ruleID)

	rule := getFilteringRule(t, ruleID)
	assert.Equal(t, createReq.Name, rule.Name)
	assert.Equal(t, createReq.Expression, rule.Expression)
	assert.Equal(t, createReq.Priority, rule.Priority)
	assert.Equal(t, *createReq.Enabled, rule.Enabled)

	rules := listFilteringRules(t)
	assert.GreaterOrEqual(t, len(rules), 1)
	found := false
	for _, r := range rules {
		if r.ID == ruleID {
			found = true
			break
		}
	}
	assert.True(t, found, "created rule should be in the list")

	updateReq := management.UpdateFilteringRuleRequest{
		Name:       stringPtr("updated_rule"),
		Expression: stringPtr("payload.status == 'inactive'"),
		Priority:   intPtr(20),
		Enabled:    boolPtr(false),
	}
	updatedRule := updateFilteringRule(t, ruleID, updateReq)
	assert.Equal(t, *updateReq.Name, updatedRule.Name)
	assert.Equal(t, *updateReq.Expression, updatedRule.Expression)
	assert.Equal(t, *updateReq.Priority, updatedRule.Priority)
	assert.Equal(t, *updateReq.Enabled, updatedRule.Enabled)

	versions := getRuleVersions(t, ruleID)
	assert.GreaterOrEqual(t, len(versions), 1)

	auditLogs := getRuleAuditLogs(t, ruleID)
	assert.GreaterOrEqual(t, len(auditLogs), 0)
}

func TestEnrichmentRulesCRUD(t *testing.T) {
	createReq := management.CreateEnrichmentRuleRequest{
		Name:          "test_enrichment_rule",
		FieldToEnrich: "user_id",
		SourceType:    "redis",
		SourceConfig: management.EnrichmentSourceConfig{
			KeyPattern: "user:{user_id}",
		},
		Priority:      10,
		Enabled:       boolPtr(true),
		ErrorHandling: "skip_field",
	}

	ruleID := createEnrichmentRule(t, createReq)
	defer deleteEnrichmentRule(t, ruleID)

	rule := getEnrichmentRule(t, ruleID)
	assert.Equal(t, createReq.Name, rule.Name)
	assert.Equal(t, createReq.FieldToEnrich, rule.FieldToEnrich)
	assert.Equal(t, createReq.SourceType, rule.SourceType)

	rules := listEnrichmentRules(t)
	assert.GreaterOrEqual(t, len(rules), 1)
	found := false
	for _, r := range rules {
		if r.ID == ruleID {
			found = true
			break
		}
	}
	assert.True(t, found, "created rule should be in the list")

	updateReq := management.UpdateEnrichmentRuleRequest{
		Name:          stringPtr("updated_enrichment_rule"),
		Priority:      intPtr(20),
		Enabled:       boolPtr(false),
		ErrorHandling: stringPtr("fail"),
	}
	updatedRule := updateEnrichmentRule(t, ruleID, updateReq)
	assert.Equal(t, *updateReq.Name, updatedRule.Name)
	assert.Equal(t, *updateReq.Priority, updatedRule.Priority)
	assert.Equal(t, *updateReq.Enabled, updatedRule.Enabled)
}

func TestDeduplicationConfig(t *testing.T) {
	updateReq := management.UpdateDeduplicationConfigRequest{
		HashAlgorithm: stringPtr("sha256"),
		TTLSeconds:    intPtr(3600),
		OnRedisError:  stringPtr("allow"),
		FieldsToHash:  &[]string{"id", "source"},
	}
	updatedConfig := updateDeduplicationConfig(t, updateReq)
	assert.Equal(t, *updateReq.HashAlgorithm, updatedConfig.HashAlgorithm)
	assert.Equal(t, *updateReq.TTLSeconds, updatedConfig.TTLSeconds)
	assert.Equal(t, *updateReq.OnRedisError, updatedConfig.OnRedisError)
	assert.Equal(t, *updateReq.FieldsToHash, updatedConfig.FieldsToHash)

	config := getDeduplicationConfig(t)
	if config.HashAlgorithm != "" {
		assert.Equal(t, *updateReq.HashAlgorithm, config.HashAlgorithm)
		assert.Equal(t, *updateReq.TTLSeconds, config.TTLSeconds)
	}
}

func TestAuditLogs(t *testing.T) {
	createReq := management.CreateFilteringRuleRequest{
		Name:       "audit_test_rule",
		Expression: "payload.status == 'active'",
		Priority:   10,
		Enabled:    boolPtr(true),
	}
	ruleID := createFilteringRule(t, createReq)
	defer deleteFilteringRule(t, ruleID)

	updateReq := management.UpdateFilteringRuleRequest{
		Name: stringPtr("updated_audit_test_rule"),
	}
	_ = updateFilteringRule(t, ruleID, updateReq)

	time.Sleep(1 * time.Second)

	auditLogs := getRuleAuditLogs(t, ruleID)
	assert.GreaterOrEqual(t, len(auditLogs), 1)

	allLogs := getAllAuditLogs(t)
	assert.GreaterOrEqual(t, len(allLogs), 1)

	filteredLogs := getAllAuditLogsWithFilter(t, "", "filtering")
	assert.GreaterOrEqual(t, len(filteredLogs), 1)
}

func TestValidationErrors(t *testing.T) {
	invalidReq := management.CreateFilteringRuleRequest{
		Name: "",
	}
	resp := createFilteringRuleWithError(t, invalidReq)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	invalidEnrichmentReq := management.CreateEnrichmentRuleRequest{
		Name: "",
	}
	resp = createEnrichmentRuleWithError(t, invalidEnrichmentReq)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func createFilteringRule(t *testing.T, req management.CreateFilteringRuleRequest) string {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/rules/filtering", managementServiceURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var rule management.FilteringRule
	err = json.NewDecoder(resp.Body).Decode(&rule)
	require.NoError(t, err)

	return rule.ID
}

func getFilteringRule(t *testing.T, id string) management.FilteringRule {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/rules/filtering/%s", managementServiceURL, id))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rule management.FilteringRule
	err = json.NewDecoder(resp.Body).Decode(&rule)
	require.NoError(t, err)

	return rule
}

func listFilteringRules(t *testing.T) []management.FilteringRule {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/rules/filtering", managementServiceURL))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rules []management.FilteringRule
	err = json.NewDecoder(resp.Body).Decode(&rules)
	require.NoError(t, err)

	return rules
}

func updateFilteringRule(t *testing.T, id string, req management.UpdateFilteringRuleRequest) management.FilteringRule {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/api/v1/rules/filtering/%s", managementServiceURL, id),
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rule management.FilteringRule
	err = json.NewDecoder(resp.Body).Decode(&rule)
	require.NoError(t, err)

	return rule
}

func deleteFilteringRule(t *testing.T, id string) {
	t.Helper()

	httpReq, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/api/v1/rules/filtering/%s", managementServiceURL, id),
		nil,
	)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func getRuleVersions(t *testing.T, id string) []management.RuleVersion {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/rules/filtering/%s/versions", managementServiceURL, id))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var versions []management.RuleVersion
	err = json.NewDecoder(resp.Body).Decode(&versions)
	require.NoError(t, err)

	return versions
}

func getRuleAuditLogs(t *testing.T, id string) []management.AuditLog {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/rules/filtering/%s/audit", managementServiceURL, id))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var logs []management.AuditLog
	err = json.NewDecoder(resp.Body).Decode(&logs)
	require.NoError(t, err)

	return logs
}

func getAllAuditLogs(t *testing.T) []management.AuditLog {
	t.Helper()
	return getAllAuditLogsWithFilter(t, "", "")
}

func getAllAuditLogsWithFilter(t *testing.T, ruleID, ruleType string) []management.AuditLog {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/audit/logs", managementServiceURL)
	if ruleID != "" {
		url += fmt.Sprintf("?rule_id=%s", ruleID)
	}
	if ruleType != "" {
		if strings.Contains(url, "?") {
			url += "&"
		} else {
			url += "?"
		}
		url += fmt.Sprintf("rule_type=%s", ruleType)
	}

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var logs []management.AuditLog
	err = json.NewDecoder(resp.Body).Decode(&logs)
	require.NoError(t, err)

	return logs
}

func createEnrichmentRule(t *testing.T, req management.CreateEnrichmentRuleRequest) string {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/rules/enrichment", managementServiceURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var rule management.EnrichmentRule
	err = json.NewDecoder(resp.Body).Decode(&rule)
	require.NoError(t, err)

	return rule.ID
}

func getEnrichmentRule(t *testing.T, id string) management.EnrichmentRule {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/rules/enrichment/%s", managementServiceURL, id))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rule management.EnrichmentRule
	err = json.NewDecoder(resp.Body).Decode(&rule)
	require.NoError(t, err)

	return rule
}

func listEnrichmentRules(t *testing.T) []management.EnrichmentRule {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/rules/enrichment", managementServiceURL))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rules []management.EnrichmentRule
	err = json.NewDecoder(resp.Body).Decode(&rules)
	require.NoError(t, err)

	return rules
}

func updateEnrichmentRule(t *testing.T, id string, req management.UpdateEnrichmentRuleRequest) management.EnrichmentRule {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/api/v1/rules/enrichment/%s", managementServiceURL, id),
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rule management.EnrichmentRule
	err = json.NewDecoder(resp.Body).Decode(&rule)
	require.NoError(t, err)

	return rule
}

func deleteEnrichmentRule(t *testing.T, id string) {
	t.Helper()

	httpReq, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/api/v1/rules/enrichment/%s", managementServiceURL, id),
		nil,
	)
	require.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func getDeduplicationConfig(t *testing.T) management.DeduplicationConfig {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/config/deduplication", managementServiceURL))
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return management.DeduplicationConfig{}
	}

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var config management.DeduplicationConfig
	err = json.NewDecoder(resp.Body).Decode(&config)
	require.NoError(t, err)

	return config
}

func updateDeduplicationConfig(t *testing.T, req management.UpdateDeduplicationConfigRequest) management.DeduplicationConfig {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/api/v1/config/deduplication", managementServiceURL),
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var config management.DeduplicationConfig
	err = json.NewDecoder(resp.Body).Decode(&config)
	require.NoError(t, err)

	return config
}

func createFilteringRuleWithError(t *testing.T, req management.CreateFilteringRuleRequest) *http.Response {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/rules/filtering", managementServiceURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)

	return resp
}

func createEnrichmentRuleWithError(t *testing.T, req management.CreateEnrichmentRuleRequest) *http.Response {
	t.Helper()

	body, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/rules/enrichment", managementServiceURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(t, err)

	return resp
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
