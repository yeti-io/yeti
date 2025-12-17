package deduplication

// Config represents deduplication configuration entity
type Config struct {
	HashAlgorithm string
	TTLSeconds    int
	OnRedisError  string
	FieldsToHash  []string
}
