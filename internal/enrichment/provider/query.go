package provider

type Query struct {
	Filters map[string]interface{} `json:"filters,omitempty"`
	Sort    map[string]interface{} `json:"sort,omitempty"`
	Limit   *int                   `json:"limit,omitempty"`
	Offset  *int                   `json:"offset,omitempty"`
}

func (q *Query) ToMap() map[string]interface{} {
	if q == nil {
		return make(map[string]interface{})
	}
	result := make(map[string]interface{})
	if q.Filters != nil {
		result["filters"] = q.Filters
	}
	if q.Sort != nil {
		result["sort"] = q.Sort
	}
	if q.Limit != nil {
		result["limit"] = *q.Limit
	}
	if q.Offset != nil {
		result["offset"] = *q.Offset
	}
	return result
}

func QueryFromMap(m map[string]interface{}) *Query {
	if m == nil {
		return &Query{Filters: make(map[string]interface{})}
	}

	q := &Query{
		Filters: make(map[string]interface{}),
	}

	if filters, ok := m["filters"].(map[string]interface{}); ok {
		q.Filters = filters
	}
	if sort, ok := m["sort"].(map[string]interface{}); ok {
		q.Sort = sort
	}
	if limit, ok := m["limit"].(int); ok {
		q.Limit = &limit
	}
	if offset, ok := m["offset"].(int); ok {
		q.Offset = &offset
	}

	return q
}
