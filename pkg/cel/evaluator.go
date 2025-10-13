package cel

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"

	"yeti/pkg/models"
)

type Evaluator struct {
	env *cel.Env
}

func NewEvaluator() (*Evaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable("id", cel.StringType),
		cel.Variable("source", cel.StringType),
		cel.Variable("timestamp", cel.TimestampType),
		cel.Variable("payload", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("metadata", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Evaluator{env: env}, nil
}

func (e *Evaluator) ValidateExpression(expression string) error {
	_, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("CEL expression validation failed: %w", issues.Err())
	}
	return nil
}

func (e *Evaluator) ValidateFilterExpression(expression string) error {
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("CEL expression validation failed: %w", issues.Err())
	}

	if ast.OutputType() != cel.BoolType {
		return fmt.Errorf("filter expression must return bool, got %v", ast.OutputType())
	}

	return nil
}

func (e *Evaluator) EvaluateFilter(ctx context.Context, expression string, msg models.MessageEnvelope) (bool, error) {
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	if ast.OutputType() != cel.BoolType {
		return false, fmt.Errorf("filter expression must return bool, got %v", ast.OutputType())
	}

	program, err := e.env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL program: %w", err)
	}

	vars := map[string]interface{}{
		"id":        msg.ID,
		"source":    msg.Source,
		"timestamp": msg.Timestamp,
		"payload":   msg.Payload,
		"metadata":  e.metadataToMap(msg.Metadata),
	}

	result, _, err := program.ContextEval(ctx, vars)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	boolVal, ok := result.Value().(bool)
	if !ok {
		return false, fmt.Errorf("CEL expression did not return bool, got %T", result.Value())
	}

	return boolVal, nil
}

func (e *Evaluator) EvaluateTransform(ctx context.Context, expression string, msg models.MessageEnvelope, sourceData map[string]interface{}) (interface{}, error) {
	env, err := cel.NewEnv(
		cel.Variable("id", cel.StringType),
		cel.Variable("source", cel.StringType),
		cel.Variable("timestamp", cel.TimestampType),
		cel.Variable("payload", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("metadata", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("sourceData", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	program, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	vars := map[string]interface{}{
		"id":         msg.ID,
		"source":     msg.Source,
		"timestamp":  msg.Timestamp,
		"payload":    msg.Payload,
		"metadata":   e.metadataToMap(msg.Metadata),
		"sourceData": sourceData,
	}

	result, _, err := program.ContextEval(ctx, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate CEL expression: %w", err)
	}

	return result.Value(), nil
}

func (e *Evaluator) metadataToMap(metadata models.Metadata) map[string]interface{} {
	result := make(map[string]interface{})

	if metadata.TraceID != "" {
		result["trace_id"] = metadata.TraceID
	}

	if metadata.FiltersApplied != nil {
		result["filters_applied"] = map[string]interface{}{
			"passed_at": metadata.FiltersApplied.PassedAt,
			"rule_ids":  metadata.FiltersApplied.RuleIDs,
		}
	}

	if metadata.Deduplication != nil {
		result["deduplication"] = map[string]interface{}{
			"is_unique":  metadata.Deduplication.IsUnique,
			"checked_at": metadata.Deduplication.CheckedAt,
		}
	}

	if metadata.Enrichment != nil {
		result["enrichment"] = metadata.Enrichment
	}

	return result
}

func (e *Evaluator) CompileExpression(expression string) (cel.Program, error) {
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	program, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	return program, nil
}
