package provider

import (
	"context"
)

type DataProvider interface {
	Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (map[string]interface{}, error)
}

type TypedDataProvider interface {
	FetchTyped(ctx context.Context, config SourceConfig, fieldValue interface{}) (*EnrichmentResult, error)
}
