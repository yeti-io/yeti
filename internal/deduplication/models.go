package deduplication

type Config struct {
	HashAlgorithm string
	TTLSeconds    int
	OnRedisError  string
	FieldsToHash  []string
}
