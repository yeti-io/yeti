package provider

type EnrichmentResult struct {
	Data map[string]interface{} `json:"data"`
}

func (r *EnrichmentResult) ToMap() map[string]interface{} {
	if r == nil {
		return make(map[string]interface{})
	}
	return r.Data
}

func EnrichmentResultFromMap(m map[string]interface{}) *EnrichmentResult {
	if m == nil {
		return &EnrichmentResult{Data: make(map[string]interface{})}
	}
	return &EnrichmentResult{Data: m}
}

func (r *EnrichmentResult) GetField(name string) (interface{}, bool) {
	if r == nil || r.Data == nil {
		return nil, false
	}
	value, ok := r.Data[name]
	return value, ok
}

func (r *EnrichmentResult) SetField(name string, value interface{}) {
	if r == nil {
		r = &EnrichmentResult{Data: make(map[string]interface{})}
	}
	if r.Data == nil {
		r.Data = make(map[string]interface{})
	}
	r.Data[name] = value
}
