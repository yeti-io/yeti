package cel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"yeti/pkg/models"
)

func TestNewEvaluator(t *testing.T) {
	eval, err := NewEvaluator()
	require.NoError(t, err)
	assert.NotNil(t, eval)
}

func TestValidateExpression(t *testing.T) {
	eval, err := NewEvaluator()
	require.NoError(t, err)

	tests := []struct {
		name      string
		expr      string
		wantError bool
	}{
		{
			name:      "valid simple expression",
			expr:      `payload.status == "active"`,
			wantError: false,
		},
		{
			name:      "valid numeric comparison",
			expr:      `payload.amount > 100.0`,
			wantError: false,
		},
		{
			name:      "invalid expression",
			expr:      `invalid syntax here!!!`,
			wantError: true,
		},
		{
			name:      "undefined variable",
			expr:      `undefinedVar == "test"`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := eval.ValidateExpression(tt.expr)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFilterExpression(t *testing.T) {
	eval, err := NewEvaluator()
	require.NoError(t, err)

	tests := []struct {
		name      string
		expr      string
		wantError bool
	}{
		{
			name:      "valid bool expression",
			expr:      `payload.status == "active"`,
			wantError: false,
		},
		{
			name:      "non-bool expression",
			expr:      `payload.amount`,
			wantError: true,
		},
		{
			name:      "valid contains",
			expr:      `payload.email.contains("@example.com")`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := eval.ValidateFilterExpression(tt.expr)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTransformExpression(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		wantError bool
	}{
		{
			name:      "valid simple expression",
			expr:      `sourceData.name`,
			wantError: false,
		},
		{
			name:      "valid concatenation",
			expr:      `sourceData.firstName + " " + sourceData.lastName`,
			wantError: false,
		},
		{
			name:      "valid conditional",
			expr:      `sourceData.status == "active" ? "enabled" : "disabled"`,
			wantError: false,
		},
		{
			name:      "valid math operation",
			expr:      `sourceData.price * 1.1`,
			wantError: false,
		},
		{
			name:      "valid contains",
			expr:      `sourceData.email.contains("@")`,
			wantError: false,
		},
		{
			name:      "test upper method",
			expr:      `sourceData.name.upper()`,
			wantError: true,
		},
		{
			name:      "test lower method",
			expr:      `sourceData.name.lower()`,
			wantError: true,
		},
		{
			name:      "test upperAscii method",
			expr:      `sourceData.name.upperAscii()`,
			wantError: false,
		},
		{
			name:      "test lowerAscii method",
			expr:      `sourceData.name.lowerAscii()`,
			wantError: false,
		},
		{
			name:      "test string functions",
			expr:      `sourceData.name.matches(".*")`,
			wantError: false,
		},
		{
			name:      "test string length",
			expr:      `sourceData.name.size()`,
			wantError: false,
		},
		{
			name:      "test string indexOf",
			expr:      `sourceData.email.indexOf("@")`,
			wantError: false,
		},
	}

	eval, err := NewEvaluator()
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := eval.ValidateTransformExpression(tt.expr)
			if tt.wantError {
				assert.Error(t, err, "Expected error for expression: %s", tt.expr)
				t.Logf("Expression '%s' failed as expected: %v", tt.expr, err)
			} else {
				assert.NoError(t, err, "Expected no error for expression: %s", tt.expr)
			}
		})
	}
}

func TestEvaluateFilter(t *testing.T) {
	eval, err := NewEvaluator()
	require.NoError(t, err)

	ctx := context.Background()
	msg := models.MessageEnvelope{
		ID:        "test-id",
		Source:    "test-source",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"status": "active",
			"amount": 150.0,
			"email":  "user@example.com",
		},
		Metadata: models.Metadata{},
	}

	tests := []struct {
		name      string
		expr      string
		want      bool
		wantError bool
	}{
		{
			name:      "simple equality true",
			expr:      `payload.status == "active"`,
			want:      true,
			wantError: false,
		},
		{
			name:      "simple equality false",
			expr:      `payload.status == "inactive"`,
			want:      false,
			wantError: false,
		},
		{
			name:      "numeric comparison true",
			expr:      `payload.amount > 100.0`,
			want:      true,
			wantError: false,
		},
		{
			name:      "numeric comparison false",
			expr:      `payload.amount > 200.0`,
			want:      false,
			wantError: false,
		},
		{
			name:      "contains true",
			expr:      `payload.email.contains("@example.com")`,
			want:      true,
			wantError: false,
		},
		{
			name:      "contains false",
			expr:      `payload.email.contains("@other.com")`,
			want:      false,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.EvaluateFilter(ctx, tt.expr, msg)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestEvaluateTransform(t *testing.T) {
	ctx := context.Background()
	msg := models.MessageEnvelope{
		ID:        "test-id",
		Source:    "test-source",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"user_id": "user-123",
		},
		Metadata: models.Metadata{},
	}

	sourceData := map[string]interface{}{
		"name":     "test product",
		"price":    99.99,
		"category": "electronics",
		"email":    "user@example.com",
		"status":   "active",
	}

	tests := []struct {
		name      string
		expr      string
		want      interface{}
		wantError bool
	}{
		{
			name:      "simple field access",
			expr:      `sourceData.name`,
			want:      "test product",
			wantError: false,
		},
		{
			name:      "string concatenation",
			expr:      `sourceData.name + " - " + sourceData.category`,
			want:      "test product - electronics",
			wantError: false,
		},
		{
			name:      "conditional expression",
			expr:      `sourceData.status == "active" ? "enabled" : "disabled"`,
			want:      "enabled",
			wantError: false,
		},
		{
			name:      "math operation",
			expr:      `sourceData.price * 1.1`,
			want:      109.989,
			wantError: false,
		},
		{
			name:      "combine payload and sourceData",
			expr:      `payload.user_id + "-" + string(sourceData.price)`,
			want:      "user-123-99.99",
			wantError: false,
		},
		{
			name:      "string size",
			expr:      `sourceData.name.size()`,
			want:      int64(12),
			wantError: false,
		},
		{
			name:      "string indexOf",
			expr:      `sourceData.email.indexOf("@")`,
			want:      int64(4),
			wantError: false,
		},
		{
			name:      "test upperAscii method",
			expr:      `sourceData.name.upperAscii()`,
			want:      "TEST PRODUCT",
			wantError: false,
		},
		{
			name:      "test lowerAscii method",
			expr:      `sourceData.name.lowerAscii()`,
			want:      "test product",
			wantError: false,
		},
	}

	eval, err := NewEvaluator()
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.EvaluateTransform(ctx, tt.expr, msg, sourceData)
			if tt.wantError {
				assert.Error(t, err, "Expected error for expression: %s", tt.expr)
				t.Logf("Expression '%s' failed as expected: %v", tt.expr, err)
			} else {
				assert.NoError(t, err, "Expected no error for expression: %s", tt.expr)
				if !tt.wantError {
					assert.Equal(t, tt.want, result, "Result mismatch for expression: %s", tt.expr)
				}
			}
		})
	}
}

func TestStringMethodsAvailable(t *testing.T) {
	ctx := context.Background()
	sourceData := map[string]interface{}{
		"name":  "Test String",
		"email": "user@example.com",
	}

	msg := models.MessageEnvelope{
		ID:        "test-id",
		Source:    "test-source",
		Timestamp: time.Now(),
		Payload:   map[string]interface{}{},
		Metadata:  models.Metadata{},
	}

	eval, err := NewEvaluator()
	require.NoError(t, err)

	stringMethods := []struct {
		name string
		expr string
	}{
		{"contains", `sourceData.email.contains("@")`},
		{"size", `sourceData.name.size()`},
		{"indexOf", `sourceData.email.indexOf("@")`},
		{"upper", `sourceData.name.upper()`},
		{"lower", `sourceData.name.lower()`},
		{"upperAscii", `sourceData.name.upperAscii()`},
		{"lowerAscii", `sourceData.name.lowerAscii()`},
		{"startsWith", `sourceData.name.startsWith("Test")`},
		{"endsWith", `sourceData.name.endsWith("String")`},
		{"matches", `sourceData.name.matches(".*")`},
		{"substring", `sourceData.name[0:4]`},
	}

	t.Log("Testing available string methods in CEL:")
	for _, method := range stringMethods {
		t.Run(method.name, func(t *testing.T) {
			transformEval, err := NewEvaluator()
			require.NoError(t, err)
			err = transformEval.ValidateTransformExpression(method.expr)
			if err != nil {
				t.Logf("  ❌ %s: validation failed - %v", method.name, err)
				return
			}

			result, err := eval.EvaluateTransform(ctx, method.expr, msg, sourceData)
			if err != nil {
				t.Logf("  ❌ %s: evaluation failed - %v", method.name, err)
				return
			}

			t.Logf("  ✓ %s: works, result = %v", method.name, result)
		})
	}
}
