package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"yeti/internal/constants"
)

type APIProvider struct {
	client *http.Client
}

func NewAPIProvider() *APIProvider {
	return &APIProvider{
		client: &http.Client{
			Timeout: constants.DefaultHTTPTimeout,
		},
	}
}

func (p *APIProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (map[string]interface{}, error) {
	url := strings.ReplaceAll(config.URL, "{field_value}", fmt.Sprintf("%v", fieldValue))
	url = strings.ReplaceAll(url, "{value}", fmt.Sprintf("%v", fieldValue))

	req, err := http.NewRequestWithContext(ctx, config.Method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < constants.HTTPStatusOKMin || resp.StatusCode >= constants.HTTPStatusOKMax {
		return nil, fmt.Errorf("api returned status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
