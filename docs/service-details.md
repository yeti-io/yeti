# 📋 Детальное описание внутреннего устройства каждого сервиса

## Оглавление
1. [Filtering Service](#filtering-service-детально)
2. [Deduplication Service](#deduplication-service-детально)
3. [Enrichment Service](#enrichment-service-детально)
4. [Management Service](#management-service-детально)

---

## Filtering Service (Детально)

### Назначение и роль
Первый этап pipeline - валидирует и фильтрует входящие сообщения согласно набору правил. Только сообщения, прошедшие все активные правила фильтрации, отправляются на следующий этап.

### Основной flow

```
Input Message
    ↓
Read from: input_events queue
    ↓
Load Rules Cache (RWMutex protected)
    ↓
For each rule in cache:
  ├─ Extract field from message
  ├─ Apply operator
  ├─ Check result
  └─ If any rule fails → FILTER OUT
    ↓
If all rules passed:
  ├─ Add metadata: filters_applied
  ├─ Publish to: dedup_events queue
  └─ Increment: messages_passed metric
    ↓
If any rule failed:
  ├─ Increment: messages_filtered metric
  ├─ Log: debug with rule_id that filtered
  └─ Message dropped
```

### Структура сервиса

```go
// internal/filtering/models.go
type Rule struct {
    ID        string      `db:"id"`           // UUID
    Name      string      `db:"name"`         // User-friendly name
    Field     string      `db:"field"`        // Message field name
    Operator  string      `db:"operator"`     // eq, contains, regex, gt, lt, in, range
    Value     interface{} `db:"value"`        // Rule value
    Priority  int         `db:"priority"`     // Execution order
    Enabled   bool        `db:"enabled"`      // Is rule active
    CreatedAt time.Time   `db:"created_at"`
    UpdatedAt time.Time   `db:"updated_at"`
    Version   int         `db:"version"`      // For optimistic locking
}

type CompiledRule struct {
    Rule      Rule
    Regex     *regexp.Regexp  // Compiled regex if operator = regex
    Compiled  bool            // Is compiled successfully
}

type FilteringService struct {
    repo       Repository              // DB access
    metrics    *metrics.Metrics        // Prometheus metrics
    rules      map[string]*CompiledRule // In-memory cache
    rulesMu    sync.RWMutex           // Thread-safe access
    logger     logger.Logger
}

type Message map[string]interface{}
```

### Основные методы

```go
// Process выполняет фильтрацию сообщения
func (fs *FilteringService) Process(ctx context.Context, msg Message) (bool, error)
// Returns: (passed, error)

// GetRules возвращает список активных правил
func (fs *FilteringService) GetRules() []Rule

// ReloadRules перезагружает правила из БД
func (fs *FilteringService) ReloadRules(ctx context.Context) error

// UpsertRule создает или обновляет правило
func (fs *FilteringService) UpsertRule(ctx context.Context, rule Rule) error

// DeleteRule удаляет правило по ID
func (fs *FilteringService) DeleteRule(ctx context.Context, ruleID string) error
```

### Операторы фильтрации

```
1. eq (equals)
   - Точное совпадение строки
   - field = "status", value = "active"
   - Message: {status: "active"} → PASS
   - Message: {status: "inactive"} → FAIL

2. contains
   - Проверка на содержание подстроки
   - field = "email", value = "@example.com"
   - Message: {email: "user@example.com"} → PASS

3. regex
   - Проверка регулярным выражением
   - field = "email", value = "^[a-zA-Z0-9._%+-]+@..."
   - Regex компилируется при загрузке правила
   - Ошибка компиляции логируется, правило пропускается

4. gt (greater than)
   - Больше (для чисел)
   - field = "amount", value = 100
   - Message: {amount: 150} → PASS
   - Message: {amount: 50} → FAIL

5. lt (less than)
   - Меньше (для чисел)
   - field = "age", value = 18
   - Message: {age: 16} → PASS

6. in (in list)
   - Значение в списке
   - field = "status", value = ["active", "pending"]
   - Message: {status: "active"} → PASS
   - Message: {status: "deleted"} → FAIL

7. range
   - Значение в диапазоне [min, max]
   - field = "amount", value = [10, 1000]
   - Message: {amount: 500} → PASS
   - Message: {amount: 5} → FAIL
   - Message: {amount: 2000} → FAIL
```

### Обработка ошибок

```
Scenario: Missing field in message
├─ config: on_missing_field = "filter_out"
├─ Action: Return false (message filtered)
└─ Metric: filtered_messages++

Scenario: Type mismatch (field is string, operator expects number)
├─ config: on_type_mismatch = "filter_out"
├─ Action: Return false
└─ Metric: filtered_messages++

Scenario: Regex compilation error
├─ Action: Log error, skip rule
├─ Next rules: Still applied
└─ Metric: errors++

Scenario: Database error while loading rules
├─ Action: Keep existing rules in cache
├─ Log: error with retry info
└─ Retry: Exponential backoff 1s, 2s, 4s (max 3)
```

### Hot reload механизм

```
Method 1: SIGHUP Signal
├─ System receives SIGHUP
├─ Handler: ReloadRules()
├─ Load from PostgreSQL
├─ Acquire RWMutex write lock
├─ Replace fs.rules map
├─ Release lock
└─ New messages use new rules

Method 2: Polling (every 60 seconds)
├─ Background goroutine
├─ Query PostgreSQL for updated_at > last_check
├─ If any rules updated
├─ Call ReloadRules()
└─ Continue polling

Method 3: Event-driven (via RabbitMQ)
├─ Subscribe to: config.updates exchange
├─ On message: filtering.rules.updated
├─ Call: ReloadRules()
└─ Process updated rules
```

### Database Schema (PostgreSQL)

```sql
-- Основная таблица правил
CREATE TABLE filtering_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL UNIQUE,
  field VARCHAR(255) NOT NULL,
  operator VARCHAR(50) NOT NULL CHECK (operator IN ('eq','contains','regex','gt','lt','in','range')),
  value JSONB NOT NULL,
  priority INTEGER DEFAULT 0,
  enabled BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by VARCHAR(255),
  updated_by VARCHAR(255),
  version INTEGER DEFAULT 1 -- Optimistic locking
);

-- Индексы
CREATE INDEX idx_filtering_rules_enabled ON filtering_rules(enabled);
CREATE INDEX idx_filtering_rules_priority ON filtering_rules(priority DESC);
CREATE INDEX idx_filtering_rules_updated_at ON filtering_rules(updated_at DESC);

-- Логирование применения правил (optional)
CREATE TABLE filtering_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id UUID REFERENCES filtering_rules(id),
  message_id VARCHAR(255) NOT NULL,
  matched BOOLEAN NOT NULL,
  processing_time_ms INTEGER,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_filtering_logs_created_at ON filtering_logs(created_at DESC);
```

### Metrics (Prometheus)

```
filtering_messages_total{status="passed|filtered|error"}
├─ Тип: Counter
└─ Описание: Всего обработано сообщений

filtering_messages_duration_ms{status="passed|filtered"}
├─ Тип: Histogram
├─ Buckets: [1, 5, 10, 50, 100, 500, 1000]
└─ Описание: Время обработки в мс

filtering_rules_total{enabled="true|false"}
├─ Тип: Gauge
└─ Описание: Всего правил (активных/неактивных)

filtering_cache_size_bytes
├─ Тип: Gauge
└─ Описание: Размер кеша правил в оперативной памяти

filtering_rule_evaluations_total{rule_id="..."}
├─ Тип: Counter
└─ Описание: Сколько раз правило было применено

filtering_rule_matches_total{rule_id="..."}
├─ Тип: Counter
└─ Описание: Сколько раз правило прошло (match=true)
```

### Пример использования

```go
// cmd/filtering-service/main.go
package main

import (
    "context"
    "data-pipeline/internal/filtering"
    "data-pipeline/internal/config"
    "data-pipeline/internal/logger"
    "data-pipeline/internal/storage"
)

func main() {
    // Инициализация
    log := logger.New("filtering-service")
    cfg := config.Load("config.base.yaml")
    
    // База данных
    db := storage.NewPostgres(cfg.Database.Postgres)
    defer db.Close()
    
    // Metrics
    metrics := metrics.New()
    
    // Сервис
    repo := filtering.NewRepository(db)
    service := filtering.NewService(repo, metrics, log)
    
    // Hot reload на SIGHUP
    config.WatchAndReload("config", service.ReloadRules)
    
    // Обработка сообщений из broker
    consumer := broker.NewConsumer(cfg.Broker)
    producer := broker.NewProducer(cfg.Broker)
    
    msgChan, _ := consumer.Consume("input_events")
    
    for msg := range msgChan {
        ctx := context.Background()
        
        // Фильтрация
        passed, err := service.Process(ctx, msg)
        
        if err != nil {
            log.Errorf("Filtering error: %v", err)
            continue
        }
        
        if passed {
            // Добавить метаданные
            msg["filters_applied"] = map[string]interface{}{
                "passed_at": time.Now(),
            }
            
            // Отправить дальше
            producer.Publish("dedup_events", msg)
        }
    }
}
```

---

## Deduplication Service (Детально)

### Назначение и роль
Второй этап pipeline - проверяет является ли сообщение дубликатом на основе хеша. Использует Redis для быстрого поиска с настраиваемым временным окном.

### Основной flow

```
Input Message (с filters_applied)
    ↓
Read from: dedup_events queue
    ↓
Extract fields for hashing:
├─ id
├─ timestamp
└─ source
    ↓
Compute hash:
└─ hash = md5(id + timestamp + source)
    ↓
Check Redis:
├─ SET key dedup:{hash}
├─ With EX (expire) = 3600 (1 hour)
├─ If SET successful → UNIQUE
└─ If SET returns nil → DUPLICATE
    ↓
If unique:
├─ Add metadata: deduplication.is_unique = true
├─ Publish to: enrichment_events queue
└─ Increment: dedup_unique_messages metric
    ↓
If duplicate:
├─ Add metadata: deduplication.is_unique = false
├─ Optional: Send to DLQ
└─ Increment: dedup_duplicate_messages metric
```

### Структура сервиса

```go
// internal/deduplication/models.go
type HashConfig struct {
    Algorithm string   `yaml:"algorithm"` // md5, sha256
    Fields    []string `yaml:"fields"`    // Fields for hash
}

type DeduplicationService struct {
    redis        *redis.Client
    window       time.Duration
    hashConfig   HashConfig
    metrics      *metrics.Metrics
    logger       logger.Logger
    
    // Statistics
    stats struct {
        mu        sync.RWMutex
        unique    int64
        duplicate int64
        errors    int64
    }
}

type Message map[string]interface{}
```

### Основные методы

```go
// Process проверяет уникальность сообщения
func (ds *DeduplicationService) Process(ctx context.Context, msg Message) (bool, error)
// Returns: (isUnique, error)

// UpdateWindow обновляет окно дедубликации (hot reload)
func (ds *DeduplicationService) UpdateWindow(ctx context.Context, window time.Duration) error

// GetStats возвращает статистику дедубликации
func (ds *DeduplicationService) GetStats() DeduplicationStats

// ClearCache очищает Redis (для тестирования)
func (ds *DeduplicationService) ClearCache(ctx context.Context) error

// ComputeHash вычисляет хеш сообщения
func (ds *DeduplicationService) ComputeHash(msg Message) (string, error)
```

### Алгоритм хеширования

```go
// Пример вычисления хеша
func (ds *DeduplicationService) ComputeHash(msg Message) (string, error) {
    // Достаем поля
    id := msg["id"]
    timestamp := msg["timestamp"]
    source := msg["source"]
    
    // Создаем строку для хеша
    hashInput := fmt.Sprintf("%v%v%v", id, timestamp, source)
    
    var hash string
    switch ds.hashConfig.Algorithm {
    case "md5":
        hash = fmt.Sprintf("%x", md5.Sum([]byte(hashInput)))
    case "sha256":
        hash = fmt.Sprintf("%x", sha256.Sum256([]byte(hashInput)))
    }
    
    return fmt.Sprintf("dedup:%s", hash), nil
}

// Redis операция
// SET dedup:abc123def456 1734268500 EX 3600
// EX 3600 = expire в 3600 секунд (1 час)
```

### Redis Schema

```
Key Pattern:        dedup:{hash}
Value:              Timestamp (Unix format)
TTL:                Configurable (default: 3600 sec)
Eviction Policy:    allkeys-lru (least recently used)

Examples:
Key:    dedup:a1b2c3d4e5f6g7h8
Value:  1734268500
TTL:    3599 (expires in 3599 seconds)

Key:    dedup:z9y8x7w6v5u4t3s2
Value:  1734268450
TTL:    3600
```

### Обработка ошибок

```
Scenario: Redis connection timeout
├─ Max retries: 3
├─ Backoff: exponential (100ms, 200ms, 400ms)
├─ After max retries: send to DLQ
└─ Log: error with traceback

Scenario: Invalid message (missing id/timestamp)
├─ Action: Log warning
├─ Behavior: Allow message (config: on_invalid_message = "allow")
└─ Metric: errors++

Scenario: Hash computation fails
├─ Action: Log error
├─ Behavior: Send to DLQ (config: error_handling = "send_dlq")
└─ Manual review required

Scenario: Redis key eviction (window expired naturally)
├─ Action: None needed
├─ Behavior: Message treated as unique if seen again
├─ Reason: Window expired, duplicate check not relevant
└─ This is expected behavior
```

### Hot reload конфигурации

```
ConfigUpdate: {
  "window_seconds": 7200,  // Изменили с 3600 на 7200
  "hash_algorithm": "sha256"
}

Flow:
1. Management API: PUT /api/v1/config/deduplication
2. Update: ds.window = 7200 seconds
3. Update: ds.hashConfig.Algorithm = "sha256"
4. New messages: использованию новый алгоритм и окно
5. Old messages: по-прежнему используют старое окно (TTL выставлено)
```

### Database Schema (Redis)

```
Redis структура не требует инициализации схемы,
однако может быть полезно отслеживать статистику:

Дополнительные ключи для статистики:
- dedup:stats:unique_messages (counter)
- dedup:stats:duplicate_messages (counter)
- dedup:stats:errors (counter)
- dedup:stats:cache_hits (counter)
- dedup:stats:cache_misses (counter)

Обновляются каждую минуту через background goroutine.
```

### Metrics (Prometheus)

```
dedup_messages_total{status="unique|duplicate|error"}
├─ Тип: Counter
└─ Описание: Всего обработано сообщений

dedup_processing_duration_ms
├─ Тип: Histogram
├─ Buckets: [1, 5, 10, 50, 100, 500]
└─ Описание: Время проверки Redis

dedup_cache_hit_rate
├─ Тип: Gauge
├─ Range: [0.0, 1.0]
└─ Описание: Соотношение попаданий к промахам

dedup_window_duration_seconds
├─ Тип: Gauge
└─ Описание: Текущее окно дедубликации

dedup_redis_errors_total
├─ Тип: Counter
└─ Описание: Ошибки при работе с Redis

dedup_cache_size_bytes
├─ Тип: Gauge
└─ Описание: Примерный размер кеша (если tracked)
```

### Пример использования

```go
// cmd/dedup-service/main.go
package main

func main() {
    // Инициализация
    log := logger.New("dedup-service")
    cfg := config.Load("config.base.yaml")
    
    // Redis
    redisClient := storage.NewRedis(cfg.Database.Redis)
    defer redisClient.Close()
    
    // Metrics
    metrics := metrics.New()
    
    // Сервис
    service := deduplication.New(
        redisClient,
        cfg.Deduplication.Window,
        cfg.Deduplication.HashConfig,
        metrics,
        log,
    )
    
    // Hot reload конфига
    config.WatchAndReload("config", func() {
        newCfg := config.Load("config.base.yaml")
        service.UpdateWindow(context.Background(), newCfg.Deduplication.Window)
    })
    
    // Обработка сообщений
    consumer := broker.NewConsumer(cfg.Broker)
    producer := broker.NewProducer(cfg.Broker)
    
    msgChan, _ := consumer.Consume("dedup_events")
    
    for msg := range msgChan {
        ctx := context.Background()
        
        // Проверка дедубликации
        isUnique, err := service.Process(ctx, msg)
        
        if err != nil {
            log.Errorf("Dedup error: %v", err)
            // Отправить в DLQ
            continue
        }
        
        if isUnique {
            // Добавить метаданные
            msg["deduplication"] = map[string]interface{}{
                "is_unique": true,
                "checked_at": time.Now(),
            }
            
            // Отправить дальше
            producer.Publish("enrichment_events", msg)
        } else {
            // Дубликат - не отправляем дальше
            log.Debug("Duplicate message filtered", "message_id", msg["id"])
        }
    }
}
```

---

## Enrichment Service (Детально)

### Назначение и роль
Третий этап pipeline - обогащает сообщения дополнительной информацией из внешних источников (API, БД, кеш). Не блокирует pipeline если обогащение не удалось.

### Основной flow

```
Input Message (с filters_applied + deduplication)
    ↓
Read from: enrichment_events queue
    ↓
Load Enrichment Rules from MongoDB
    ↓
For each rule (in priority order):
    ├─ Extract required field from message
    ├─ Determine source type
    ├─ Check Cache (Redis)
    │   ├─ If hit: use cached data
    │   └─ If miss: fetch from source
    ├─ Fetch data:
    │   ├─ If API: HTTP call with timeout
    │   ├─ If DB: MongoDB/PostgreSQL query
    │   ├─ If Cache: Redis get
    │   └─ If File: Load from disk
    ├─ Transform data (if rules exist)
    ├─ Merge with message
    ├─ Cache result (with TTL)
    └─ Continue to next rule (even if failed)
    ↓
Add enrichment metadata
    ├─ rules_applied: [list of rule IDs]
    ├─ enriched_at: timestamp
    └─ cache_hits: count
    ↓
Publish to: processed_events queue
```

### Структура сервиса

```go
// internal/enrichment/models.go
type EnrichmentRule struct {
    ID                    string      `bson:"_id,omitempty"`
    Name                  string      `bson:"name"`
    FieldToEnrich         string      `bson:"field_to_enrich"`
    SourceType            string      `bson:"source_type"` // api, database, cache, file
    SourceConfig          SourceConfig
    TransformationRules   []Transformation
    CacheTTLSeconds       int
    ErrorHandling         string      // skip_field, skip_rule, fail
    FallbackValue         interface{}
    Priority              int
    Enabled               bool
    CreatedAt             time.Time
    UpdatedAt             time.Time
}

type SourceConfig struct {
    // API source
    URL             string
    Method          string
    TimeoutMs       int
    RetryCount      int
    Headers         map[string]string
    
    // Database source
    Database        string
    Collection      string
    Query           map[string]interface{}
    Projection      map[string]interface{}
    Limit           int
    
    // Cache source
    KeyPattern      string
    CacheType       string // redis, memcached
    
    // File source
    FilePath        string
    Format          string // json, yaml, csv
}

type Transformation struct {
    SourcePath  string      // JSON path in response
    TargetField string      // Field in message
    Transform   string      // identity, upper, lower, json_parse
    Default     interface{} // Fallback value if source missing
}

type EnrichmentService struct {
    rules         map[string]*EnrichmentRule
    rulesMu       sync.RWMutex
    providers     map[string]DataProvider  // API, DB, Cache providers
    cache         *redis.Client
    metrics       *metrics.Metrics
    logger        logger.Logger
}

type DataProvider interface {
    Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (interface{}, error)
}
```

### Основные методы

```go
// Process обогащает сообщение
func (es *EnrichmentService) Process(ctx context.Context, msg Message) (*Message, error)
// Returns: (*enrichedMessage, error)

// GetRules возвращает список активных правил
func (es *EnrichmentService) GetRules() []EnrichmentRule

// ReloadRules перезагружает правила из MongoDB
func (es *EnrichmentService) ReloadRules(ctx context.Context) error

// UpsertRule создает или обновляет правило
func (es *EnrichmentService) UpsertRule(ctx context.Context, rule EnrichmentRule) error

// FetchData получает данные из источника с кешем
func (es *EnrichmentService) FetchData(ctx context.Context, rule EnrichmentRule, fieldValue interface{}) (interface{}, error)

// ClearCache очищает кеш результатов обогащения
func (es *EnrichmentService) ClearCache(ctx context.Context) error
```

### Data Providers

```go
// 1. API Provider
type APIProvider struct {
    httpClient *http.Client
}

func (p *APIProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (interface{}, error) {
    // Заменяем {field_name} в URL на значение
    // Пример: "https://api/users/{user_id}" + user_id="123" → "https://api/users/123"
    url := strings.ReplaceAll(config.URL, "{field_value}", fmt.Sprintf("%v", fieldValue))
    
    // HTTP запрос с timeout
    req, _ := http.NewRequestWithContext(ctx, config.Method, url, nil)
    req.Header = http.Header(config.Headers)
    
    resp, err := p.httpClient.Do(req)
    // Parse response JSON
    // Return data
}

// 2. Database Provider
type DatabaseProvider struct {
    mongoClient *mongo.Client
    postgresDB  *sql.DB
}

func (p *DatabaseProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (interface{}, error) {
    // Replace placeholders in query
    query := replaceQueryPlaceholders(config.Query, fieldValue)
    
    // Execute query
    // Return result
}

// 3. Cache Provider
type CacheProvider struct {
    redisClient *redis.Client
}

func (p *CacheProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (interface{}, error) {
    // Build key from pattern
    key := strings.ReplaceAll(config.KeyPattern, "{value}", fmt.Sprintf("%v", fieldValue))
    
    // Get from Redis
    data, err := p.redisClient.Get(ctx, key).Result()
    // Return parsed data
}

// 4. File Provider
type FileProvider struct{}

func (p *FileProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (interface{}, error) {
    // Load file (cached in memory)
    // Search for entry matching fieldValue
    // Return entry
}
```

### Кеширование результатов

```
Ключ кеша:       enrich:{rule_id}:{field_value_hash}
Значение:        JSON результат обогащения
TTL:             Из конфига правила (default: 1800 sec)

Пример:
Правило: enrich-user-profile
Field:   user_id = "user-789"

Cache Key:   enrich:enrich-user-profile:abc123def456
Cache Value: {
  "name": "John Doe",
  "account_age_days": 365,
  "lifetime_value": 5000.00
}
TTL:         1800 seconds

При следующем обогащении того же user_id:
- Получить из кеша (1-5 мс вместо 50-100 мс на API)
- Обновить время жизни
- Инкрементировать метрику cache_hits
```

### Обработка ошибок

```
Scenario: API timeout
├─ Retry: 3 раза с exponential backoff
├─ After retries: apply error_handling strategy
├─ Strategy: skip_field → оставить поле пустым
│ │ skip_rule → не применять остальные трансформации
│ └ fail → отправить в DLQ
└─ Log: error с URL и timeout info

Scenario: Database query fails
├─ Action: Log error with query details
├─ Apply: error_handling strategy (skip_field default)
└─ Continue: pipeline не блокируется

Scenario: JSON parse fails
├─ Action: Log error with JSON sample
├─ Apply: fallback_value (если задан)
└─ Continue: pipeline не блокируется

Scenario: Cache miss + API failure
├─ Action: Check fallback_value
├─ If exists: use fallback
├─ If not: skip enrichment
└─ Continue: pipeline не блокируется
```

### Transformation Functions

```
identity:      Return value as-is
upper:         Convert to uppercase string
lower:         Convert to lowercase string
to_int:        Parse to integer
to_float:      Parse to float
to_bool:       Parse to boolean
json_parse:    Parse JSON string to object
json_stringify: Convert object to JSON string
concat:        Concatenate multiple values
date_format:   Format date (supports custom patterns)
truncate:      Truncate string (max length)
trim:          Remove whitespace
replace:       Replace substring
split:         Split string to array
join:          Join array to string
```

### Database Schema (MongoDB)

```javascript
// Коллекция правил обогащения
db.enrichment_rules.createCollection("enrichment_rules", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["name", "source_type", "enabled"],
      properties: {
        _id: { bsonType: "objectId" },
        name: { bsonType: "string" },
        field_to_enrich: { bsonType: "string" },
        source_type: { enum: ["api", "database", "cache", "file"] },
        source_config: { bsonType: "object" },
        transformation_rules: { bsonType: "array" },
        cache_ttl_seconds: { bsonType: "int" },
        error_handling: { enum: ["skip_field", "skip_rule", "fail"] },
        fallback_value: {},  // Любой тип
        priority: { bsonType: "int" },
        enabled: { bsonType: "bool" },
        created_at: { bsonType: "date" },
        updated_at: { bsonType: "date" }
      }
    }
  }
});

// Индексы
db.enrichment_rules.createIndex({ "enabled": 1, "priority": 1 });
db.enrichment_rules.createIndex({ "updated_at": -1 });

// Коллекция для кеша обогащения
db.enrichment_cache.createCollection("enrichment_cache");

// TTL индекс (автоматически удаляет документы через N секунд)
db.enrichment_cache.createIndex(
  { "created_at": 1 },
  { expireAfterSeconds: 3600 }
);

// Структура документа в кеше:
{
  _id: ObjectId(),
  rule_id: "enrich-user-profile",
  field_value_hash: "abc123def456",
  data: { ... },
  created_at: ISODate("2025-12-14T14:55:00Z")
}
```

### Metrics (Prometheus)

```
enrichment_messages_total{status="processed|partial|error"}
├─ Тип: Counter
└─ Описание: Всего обработано сообщений

enrichment_processing_duration_ms
├─ Тип: Histogram
├─ Buckets: [1, 5, 10, 50, 100, 200, 500, 1000, 2000]
└─ Описание: Время обработки (включая все API вызовы)

enrichment_cache_hit_rate
├─ Тип: Gauge
├─ Range: [0.0, 1.0]
└─ Описание: Соотношение попаданий кеша

enrichment_rule_executions_total{rule_id="..."}
├─ Тип: Counter
└─ Описание: Сколько раз правило применено

enrichment_rule_success_total{rule_id="..."}
├─ Тип: Counter
└─ Описание: Сколько раз правило успешно применено

enrichment_api_calls_total{endpoint="..."}
├─ Тип: Counter
└─ Описание: Всего API вызовов

enrichment_api_errors_total{endpoint="..."}
├─ Тип: Counter
└─ Описание: API ошибки

enrichment_api_latency_ms{endpoint="..."}
├─ Тип: Histogram
├─ Buckets: [10, 50, 100, 500, 1000, 2000, 5000]
└─ Описание: Латенция API вызовов
```

### Пример использования

```go
// cmd/enrichment-service/main.go
package main

func main() {
    log := logger.New("enrichment-service")
    cfg := config.Load("config.base.yaml")
    
    // MongoDB для правил
    mongoClient := storage.NewMongoDB(cfg.Database.MongoDB)
    defer mongoClient.Disconnect(context.Background())
    
    // Redis для кеша
    redisClient := storage.NewRedis(cfg.Database.Redis)
    defer redisClient.Close()
    
    // Metrics
    metrics := metrics.New()
    
    // Создаем провайдеров
    providers := map[string]enrichment.DataProvider{
        "api":      enrichment.NewAPIProvider(),
        "database": enrichment.NewDatabaseProvider(mongoClient),
        "cache":    enrichment.NewCacheProvider(redisClient),
        "file":     enrichment.NewFileProvider(),
    }
    
    // Сервис обогащения
    service := enrichment.New(
        mongoClient,
        redisClient,
        providers,
        metrics,
        log,
    )
    
    // Hot reload правил
    config.WatchAndReload("config", func() {
        if err := service.ReloadRules(context.Background()); err != nil {
            log.Errorf("Failed to reload enrichment rules: %v", err)
        }
    })
    
    // Обработка сообщений
    consumer := broker.NewConsumer(cfg.Broker)
    producer := broker.NewProducer(cfg.Broker)
    
    msgChan, _ := consumer.Consume("enrichment_events")
    
    for msg := range msgChan {
        ctx := context.Background()
        
        // Обогащение
        enrichedMsg, err := service.Process(ctx, msg)
        
        if err != nil {
            log.Errorf("Enrichment error: %v", err)
            // Отправить в DLQ и continue
            continue
        }
        
        // Отправить обогащенное сообщение
        producer.Publish("processed_events", enrichedMsg)
    }
}
```

---

## Management Service (Детально)

### Назначение и роль
REST API для управления всеми правилами и конфигурацией других сервисов. Позволяет администраторам изменять правила в runtime без перезагрузки.

### Основной flow

```
HTTP Request: POST /api/v1/rules/filtering
    ↓
Validate request:
├─ Authentication
├─ Authorization (RBAC)
├─ Rule validation
└─ Business logic checks
    ↓
If validation fails:
└─ Return 400/401/403 с error details
    ↓
If validation passes:
├─ Begin transaction
├─ Insert into PostgreSQL
├─ Create version entry
├─ Create audit log entry
├─ Commit transaction
├─ If transaction fails: rollback + return 500
    ↓
Notify services:
├─ Send message to RabbitMQ: config.updates
├─ Or HTTP webhook (if configured)
    ↓
Cache invalidation:
└─ Clear HTTP response cache
    ↓
Return 201 Created с созданным правилом
```

### Структура сервиса

```go
// internal/management/models.go
type CreateRuleRequest struct {
    Name  string      `json:"name" binding:"required"`
    Field string      `json:"field" binding:"required"`
    // ... other fields
}

type UpdateRuleRequest struct {
    Name     string      `json:"name"`
    Value    interface{} `json:"value"`
    Enabled  *bool       `json:"enabled"`
    Priority *int        `json:"priority"`
    // ... other optional fields
}

type RuleResponse struct {
    ID        string      `json:"id"`
    Name      string      `json:"name"`
    CreatedAt time.Time   `json:"created_at"`
    UpdatedAt time.Time   `json:"updated_at"`
    Version   int         `json:"version"`
    // ... other fields
}

type ManagementService struct {
    filteringRepo    FilteringRepository
    dedupRepo        DedupRepository
    enrichmentRepo   EnrichmentRepository
    notifier         Notifier
    validator        RuleValidator
    metrics          *metrics.Metrics
    logger           logger.Logger
}

type Notifier interface {
    NotifyRuleCreated(ctx context.Context, rule interface{}) error
    NotifyRuleUpdated(ctx context.Context, rule interface{}) error
    NotifyRuleDeleted(ctx context.Context, ruleID string) error
}
```

### REST API Endpoints (Полный список)

```
=== FILTERING RULES ===
GET     /api/v1/rules/filtering
        Список всех правил фильтрации
        Params: enabled=true|false, page=1, limit=20
        Response: List[Rule]

POST    /api/v1/rules/filtering
        Создать новое правило
        Body: CreateRuleRequest
        Response: 201 Created, Rule

GET     /api/v1/rules/filtering/:id
        Получить правило по ID
        Response: Rule

PUT     /api/v1/rules/filtering/:id
        Обновить правило
        Body: UpdateRuleRequest
        Response: 200 OK, Rule

DELETE  /api/v1/rules/filtering/:id
        Удалить правило
        Response: 204 No Content

PATCH   /api/v1/rules/filtering/:id/toggle
        Включить/отключить правило
        Body: { "enabled": true|false }
        Response: 200 OK, Rule

GET     /api/v1/rules/filtering/:id/audit
        История изменений правила
        Response: List[AuditLog]

=== DEDUPLICATION CONFIG ===
GET     /api/v1/config/deduplication
        Текущая конфигурация дедубликации
        Response: DeduplicationConfig

PUT     /api/v1/config/deduplication
        Обновить конфигурацию
        Body: UpdateDedupConfigRequest
        Response: 200 OK, DeduplicationConfig

GET     /api/v1/stats/deduplication
        Статистика дедубликации
        Response: DedupStats

=== ENRICHMENT RULES ===
GET     /api/v1/rules/enrichment
        Список всех правил обогащения
        Response: List[EnrichmentRule]

POST    /api/v1/rules/enrichment
        Создать новое правило
        Body: CreateEnrichmentRuleRequest
        Response: 201 Created, EnrichmentRule

GET     /api/v1/rules/enrichment/:id
        Получить правило по ID
        Response: EnrichmentRule

PUT     /api/v1/rules/enrichment/:id
        Обновить правило
        Body: UpdateEnrichmentRuleRequest
        Response: 200 OK, EnrichmentRule

DELETE  /api/v1/rules/enrichment/:id
        Удалить правило
        Response: 204 No Content

PATCH   /api/v1/rules/enrichment/:id/toggle
        Включить/отключить правило
        Response: 200 OK, EnrichmentRule

=== HEALTH & STATS ===
GET     /api/v1/health
        Health check всех сервисов
        Response: HealthStatus

GET     /api/v1/stats
        Глобальная статистика pipeline
        Response: PipelineStats

GET     /api/v1/stats/pipeline
        Детальная статистика по этапам
        Response: DetailedPipelineStats

GET     /api/v1/metrics
        Prometheus метрики
        Response: text/plain (Prometheus format)

=== CONFIGURATION MANAGEMENT ===
POST    /api/v1/rules/reload-signal
        Отправить SIGHUP сигнал сервисам
        Query: service=filtering|dedup|enrichment|all
        Response: 200 OK, { "status": "signal_sent" }

GET     /api/v1/config/services
        Статус всех сервисов
        Response: { services: [ServiceStatus] }
```

### Валидация правил

```go
// Filtering Rule Validation
type FilteringRuleValidator struct{}

func (v *FilteringRuleValidator) Validate(rule Rule) error {
    if rule.Field == "" {
        return errors.New("field is required")
    }
    
    validOperators := []string{"eq", "contains", "regex", "gt", "lt", "in", "range"}
    if !contains(validOperators, rule.Operator) {
        return fmt.Errorf("invalid operator: %s", rule.Operator)
    }
    
    if rule.Value == nil {
        return errors.New("value is required")
    }
    
    // Type validation based on operator
    switch rule.Operator {
    case "gt", "lt":
        if _, ok := toFloat(rule.Value); !ok {
            return errors.New("value must be numeric for gt/lt operators")
        }
    case "regex":
        if _, err := regexp.Compile(rule.Value.(string)); err != nil {
            return fmt.Errorf("invalid regex: %v", err)
        }
    case "in":
        if _, ok := rule.Value.([]interface{}); !ok {
            return errors.New("value must be array for in operator")
        }
    case "range":
        if arr, ok := rule.Value.([]interface{}); !ok || len(arr) != 2 {
            return errors.New("value must be array of 2 elements for range operator")
        }
    }
    
    return nil
}
```

### Request/Response примеры

```http
--- CREATE FILTERING RULE ---
POST /api/v1/rules/filtering
Content-Type: application/json
Authorization: Bearer eyJ0eXAi...

{
  "name": "Premium users only",
  "field": "subscription_tier",
  "operator": "eq",
  "value": "premium",
  "priority": 1,
  "enabled": true
}

Response 201 Created:
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Premium users only",
  "field": "subscription_tier",
  "operator": "eq",
  "value": "premium",
  "priority": 1,
  "enabled": true,
  "created_at": "2025-12-14T14:55:00Z",
  "updated_at": "2025-12-14T14:55:00Z",
  "version": 1,
  "created_by": "admin@example.com"
}

--- UPDATE ENRICHMENT RULE ---
PUT /api/v1/rules/enrichment/550e8400-e29b-41d4-a716-446655440001
Content-Type: application/json
Authorization: Bearer eyJ0eXAi...

{
  "source_config": {
    "url": "https://api.example.com/users/{user_id}",
    "timeout_ms": 8000,
    "retry_count": 5
  },
  "cache_ttl_seconds": 1800
}

Response 200 OK:
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "Enrich with user profile",
  "field_to_enrich": "user_id",
  "source_type": "api",
  "source_config": { ... },
  "cache_ttl_seconds": 1800,
  "version": 2,
  "updated_at": "2025-12-14T14:56:00Z",
  "updated_by": "admin@example.com"
}

--- GET PIPELINE STATS ---
GET /api/v1/stats/pipeline

Response 200 OK:
{
  "timestamp": "2025-12-14T14:56:00Z",
  "total_messages": 1000000,
  "filtering": {
    "input": 1000000,
    "passed": 850000,
    "filtered": 150000,
    "pass_rate": 0.85,
    "error_rate": 0.001,
    "avg_processing_ms": 2.5
  },
  "deduplication": {
    "input": 850000,
    "unique": 800000,
    "duplicate": 50000,
    "unique_rate": 0.941,
    "error_rate": 0.0,
    "avg_processing_ms": 1.2
  },
  "enrichment": {
    "input": 800000,
    "enriched": 790000,
    "partial": 10000,
    "error_rate": 0.002,
    "cache_hit_rate": 0.75,
    "avg_processing_ms": 45.3
  },
  "uptime_seconds": 86400
}
```

### Database Schema (PostgreSQL)

```sql
-- Таблица версионирования правил
CREATE TABLE rule_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id UUID NOT NULL,
  rule_type VARCHAR(50) NOT NULL,  -- filtering, enrichment, dedup
  rule_data JSONB NOT NULL,
  version INTEGER NOT NULL,
  changed_by VARCHAR(255),
  change_reason TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(rule_id, version),
  FOREIGN KEY (rule_id) REFERENCES filtering_rules(id) ON DELETE CASCADE
);

-- Таблица аудит логов
CREATE TABLE rule_audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rule_id UUID,
  rule_type VARCHAR(50) NOT NULL,
  action VARCHAR(50) NOT NULL,  -- create, update, delete
  old_value JSONB,
  new_value JSONB,
  changed_by VARCHAR(255) NOT NULL,
  change_reason TEXT,
  timestamp TIMESTAMP DEFAULT NOW(),
  ip_address VARCHAR(45)
);

-- API access logs (optional)
CREATE TABLE api_access_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id VARCHAR(255),
  method VARCHAR(10),
  path VARCHAR(512),
  status_code INTEGER,
  response_time_ms INTEGER,
  request_id VARCHAR(255),
  timestamp TIMESTAMP DEFAULT NOW()
);

-- Индексы
CREATE INDEX idx_rule_versions_rule_id ON rule_versions(rule_id);
CREATE INDEX idx_rule_versions_created_at ON rule_versions(created_at DESC);
CREATE INDEX idx_audit_logs_rule_id ON rule_audit_logs(rule_id);
CREATE INDEX idx_audit_logs_timestamp ON rule_audit_logs(timestamp DESC);
CREATE INDEX idx_api_logs_user_id ON api_access_logs(user_id);
CREATE INDEX idx_api_logs_timestamp ON api_access_logs(timestamp DESC);
```

### Уведомление сервисов

```go
// Способ 1: RabbitMQ Event
type ConfigUpdateNotification struct {
    RuleID    string
    RuleType  string  // filtering, enrichment, dedup
    Action    string  // created, updated, deleted
    Timestamp time.Time
    ChangedBy string
}

// Сообщение отправляется в exchange: config.updates
// Routing key: {rule_type}.{action}
// Пример: filtering.updated, enrichment.created

// Способ 2: HTTP Webhook
type WebhookPayload struct {
    Event       string
    Rule        interface{}
    Timestamp   time.Time
    SignedHash  string  // HMAC для верификации
}

// Сервис слушает GET request на configured webhook URL
// И перезагружает rules после получения
```

### Metrics (Prometheus)

```
management_api_requests_total{method="GET|POST|PUT|DELETE",path="..."}
├─ Тип: Counter
└─ Описание: Всего API запросов

management_api_response_time_ms{path="..."}
├─ Тип: Histogram
├─ Buckets: [1, 5, 10, 50, 100, 200, 500, 1000]
└─ Описание: Время ответа API

management_rules_total{type="filtering|enrichment"}
├─ Тип: Gauge
└─ Описание: Всего правил по типам

management_rule_changes_total{action="create|update|delete"}
├─ Тип: Counter
└─ Описание: Всего изменений правил

management_api_errors_total{status="400|401|403|500"}
├─ Тип: Counter
└─ Описание: Ошибки API по кодам

management_database_transactions_total{status="commit|rollback"}
├─ Тип: Counter
└─ Описание: Транзакции БД
```

---

