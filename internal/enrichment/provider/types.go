package provider

type SourceConfig struct {
	URL        string
	Method     string
	Headers    map[string]string
	TimeoutMs  int
	RetryCount int

	Database   string
	Collection string
	Query      *Query
	Field      string

	KeyPattern string
	CacheType  string
}
