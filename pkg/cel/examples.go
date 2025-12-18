package cel

var FilterExpressionExamples = map[string]string{
	"simple_equals":        `payload.status == "active"`,
	"simple_not_equals":    `payload.status != "inactive"`,
	"numeric_greater_than": `payload.amount > 100.0`,
	"numeric_less_than":    `payload.amount < 1000.0`,
	"string_contains":      `payload.email.contains("@example.com")`,
	"in_list":              `payload.status in ["active", "pending", "processing"]`,
	"range_check":          `payload.amount >= 10.0 && payload.amount <= 10000.0`,
	"combined_conditions":  `payload.status == "active" && payload.amount > 100.0 && payload.country == "US"`,
	"nested_field":         `payload.user.tier == "premium"`,
	"top_level_source":     `source == "api-gateway"`,
	"has_field":            `has(payload.email) && payload.email != ""`,
	"complex_logic":        `(payload.status == "active" || payload.status == "pending") && payload.amount > 50.0`,
}

// TransformExpressionExamples provides example CEL expressions for transformations
var TransformExpressionExamples = map[string]string{
	"uppercase":            `sourceData.name.upperAscii()`,
	"lowercase":            `sourceData.name.lowerAscii()`,
	"concatenate":          `sourceData.firstName + " " + sourceData.lastName`,
	"math_operation":       `sourceData.price * (1.0 + sourceData.taxRate / 100.0)`,
	"conditional":          `sourceData.status == "active" ? "enabled" : "disabled"`,
	"substring":            `sourceData.email[0:sourceData.email.indexOf("@")]`,
	"default_value":        `has(sourceData.name) ? sourceData.name : "Unknown"`,
	"format_number":        `string(sourceData.amount) + " USD"`,
	"extract_from_payload": `payload.user_id + "-" + string(sourceData.id)`,
}
