# 🔧 Конфигурация сервисов

## Оглавление
1. [Base Configuration](#base-configuration)
2. [Filtering Service Config](#filtering-service-config)
3. [Deduplication Service Config](#deduplication-service-config)
4. [Enrichment Service Config](#enrichment-service-config)
5. [Management Service Config](#management-service-config)
6. [Environment Variables](#environment-variables)

---

## Base Configuration

### config/config.base.yaml
```yaml
# Data Pipeline - Base Configuration
# Этот файл содержит базовые параметры для всех сервисов

version: "1.0"
environment: "development"

# Logging configuration
logging:
  level: "info"           # debug, info, warn, error, fatal
  format: "json"          # json или text
  output: "stdout"        # stdout или file path
  file:
    max_size_mb: 100
    max_backups: 5
    max_age_days: 30
    compress: true

# Server configuration
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout_seconds: 30
  write_timeout_seconds: 30
  idle_timeout_seconds: 120
  max_header_bytes: 1048576

# Message Broker configuration
broker:
  type: "rabbitmq"        # rabbitmq, kafka, nats
  
  rabbitmq:
    host: "${RABBITMQ_HOST:localhost}"
    port: ${RABBITMQ_PORT:5672}
    username: "${RABBITMQ_USER:guest}"
    password: "${RABBITMQ_PASSWORD:guest}"
    vhost: "/"
    connection_timeout_seconds: 10
    heartbeat_seconds: 60
    prefetch_count: 10
    
    queues:
      input_events:
        name: "input_events"
        durable: true
        auto_delete: false
        dead_letter_exchange: "dlx"
        
      dedup_events:
        name: "dedup_events"
        durable: true
        auto_delete: false
        dead_letter_exchange: "dlx"
        
      enrichment_events:
        name: "enrichment_events"
        durable: true
        auto_delete: false
        dead_letter_exchange: "dlx"
        
      processed_events:
        name: "processed_events"
        durable: true
        auto_delete: false
        dead_letter_exchange: "dlx"
    
    exchanges:
      events_direct:
        name: "events.direct"
        type: "direct"
        durable: true
        
      events_fanout:
        name: "events.fanout"
        type: "fanout"
        durable: true
        
      dlx:
        name: "dlx"
        type: "direct"
        durable: true

# Database configuration
database:
  postgres:
    host: "${POSTGRES_HOST:localhost}"
    port: ${POSTGRES_PORT:5432}
    user: "${POSTGRES_USER:admin}"
    password: "${POSTGRES_PASSWORD:password}"
    dbname: "${POSTGRES_DB:filtering}"
    sslmode: "disable"  # disable, require, verify-ca, verify-full
    
    connection_pool:
      max_open_conns: 25
      max_idle_conns: 5
      conn_max_lifetime_seconds: 300
    
    query_timeout_seconds: 30
  
  redis:
    host: "${REDIS_HOST:localhost}"
    port: ${REDIS_PORT:6379}
    password: "${REDIS_PASSWORD:}"
    db: 0
    
    connection_pool:
      max_retries: 3
      pool_size: 10
    
    command_timeout_seconds: 10
  
  mongodb:
    uri: "${MONGODB_URI:mongodb://admin:password@localhost:27017}"
    database: "${MONGODB_DB:enrichment}"
    
    connection_pool:
      max_pool_size: 50
      min_pool_size: 10
    
    command_timeout_seconds: 30

# Metrics configuration (Prometheus)
metrics:
  enabled: true
  port: 8080
  path: "/metrics"
  
  collectors:
    enable_process_metrics: true
    enable_go_metrics: true
    enable_custom_metrics: true

# Health check configuration
health:
  enabled: true
  path: "/health"
  check_interval_seconds: 10
  
  checks:
    database: true
    broker: true
    cache: true

# Tracing configuration (optional)
tracing:
  enabled: false
  jaeger:
    endpoint: "http://localhost:14268/api/traces"
    service_name: "data-pipeline"
    sampler:
      type: "const"
      param: 1.0

# Common timeouts
timeouts:
  api_call_seconds: 30
  database_query_seconds: 30
  broker_operation_seconds: 30
  cache_operation_seconds: 10
  
# Retry policy
retry:
  max_attempts: 3
  initial_backoff_ms: 100
  max_backoff_ms: 10000
  backoff_multiplier: 2.0
  
# Circuit breaker
circuit_breaker:
  enabled: true
  failure_threshold: 5
  success_threshold: 2
  timeout_seconds: 60
```

---

## Filtering Service Config

### config/config.filtering.yaml
```yaml
# Filtering Service Specific Configuration

service:
  name: "filtering-service"
  version: "1.0.0"
  
filtering:
  # Правила фильтрации
  rules:
    # Пример правила 1: Фильтр по статусу
    - id: "filter-status-active"
      name: "Filter only active status"
      field: "status"
      operator: "eq"
      value: "active"
      priority: 1
      enabled: true
    
    # Пример правила 2: Фильтр по типу события
    - id: "filter-event-purchase"
      name: "Filter purchase events"
      field: "event_type"
      operator: "in"
      value: ["purchase", "order", "transaction"]
      priority: 2
      enabled: true
    
    # Пример правила 3: Фильтр по валидности email
    - id: "filter-valid-email"
      name: "Filter valid email addresses"
      field: "email"
      operator: "regex"
      value: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
      priority: 3
      enabled: true
    
    # Пример правила 4: Фильтр по минимальной сумме
    - id: "filter-amount-min"
      name: "Filter minimum amount"
      field: "amount"
      operator: "gt"
      value: 10.00
      priority: 4
      enabled: true
    
    # Пример правила 5: Фильтр по диапазону
    - id: "filter-amount-range"
      name: "Filter amount range"
      field: "amount"
      operator: "range"
      value: [10, 10000]
      priority: 5
      enabled: true
  
  # Конфигурация кеша правил
  cache:
    enabled: true
    reload_on_startup: true  # Загрузить правила из БД при старте
    background_sync_interval_seconds: 300  # Переполнять каждые 5 минут
  
  # Hot reload конфигурация
  hot_reload:
    enabled: true
    method: "sighup"  # sighup, polling, event-driven
    polling_interval_seconds: 60  # Если используется polling
    
  # Обработка ошибок
  error_handling:
    on_missing_field: "filter_out"  # filter_out или allow
    on_type_mismatch: "filter_out"
    on_regex_error: "filter_out"

# Метрики для фильтрации
metrics:
  filtering:
    enabled: true
    prefix: "filtering"
    
    counters:
      - name: "messages_total"
        help: "Total messages processed"
      - name: "messages_passed"
        help: "Messages passed filtering"
      - name: "messages_filtered"
        help: "Messages filtered out"
      - name: "rules_applied"
        help: "Rules applied count"
      - name: "errors_total"
        help: "Total errors"
    
    histograms:
      - name: "processing_duration_ms"
        help: "Processing duration in milliseconds"
        buckets: [1, 5, 10, 50, 100, 500, 1000]
    
    gauges:
      - name: "active_rules"
        help: "Number of active rules"
      - name: "cache_size"
        help: "Rules cache size"

# Логирование для фильтрации
logging:
  filtering:
    level: "info"
    enabled_fields:
      - message_id
      - rule_id
      - passed
      - processing_time_ms
```

---

## Deduplication Service Config

### config/config.dedup.yaml
```yaml
# Deduplication Service Specific Configuration

service:
  name: "dedup-service"
  version: "1.0.0"

deduplication:
  # Основные параметры дедубликации
  window:
    duration_seconds: 3600        # 1 час по умолчанию
    unit: "seconds"
  
  # Поля для вычисления хеша
  hash_config:
    algorithm: "md5"              # md5 или sha256
    fields:
      - "id"
      - "timestamp"
      - "source"
    # Опциональные дополнительные поля для специфических типов сообщений
    field_overrides:
      purchase_events:
        - "user_id"
        - "product_id"
        - "timestamp"
      user_events:
        - "user_id"
        - "timestamp"
        - "event_type"
  
  # Redis кешь конфигурация
  cache:
    enabled: true
    key_prefix: "dedup"
    ttl_seconds: 3600
    
    # Настройки для отслеживания статистики
    track_statistics: true
    stats_update_interval_seconds: 60
  
  # Обработка ошибок
  error_handling:
    on_redis_error: "allow"          # allow или filter_out
    on_invalid_message: "allow"
    fallback_strategy: "allow"       # allow или deny
    
    # Retry для Redis операций
    retry:
      max_attempts: 3
      backoff_type: "exponential"
  
  # Hot reload конфигурация
  hot_reload:
    enabled: true
    fields_support_dynamic_update: true  # Можно менять поля для хеша runtime

# Метрики для дедубликации
metrics:
  deduplication:
    enabled: true
    prefix: "dedup"
    
    counters:
      - name: "messages_total"
        help: "Total messages processed"
      - name: "unique_messages"
        help: "Unique messages"
      - name: "duplicate_messages"
        help: "Duplicate messages"
      - name: "cache_misses"
        help: "Cache misses"
      - name: "cache_errors"
        help: "Cache operation errors"
    
    gauges:
      - name: "cache_size"
        help: "Cache size in bytes"
      - name: "cache_hit_rate"
        help: "Cache hit rate (0-1)"
      - name: "window_duration_seconds"
        help: "Current dedup window duration"
    
    histograms:
      - name: "processing_duration_ms"
        help: "Processing duration in milliseconds"
        buckets: [1, 5, 10, 50, 100, 500]

# Логирование для дедубликации
logging:
  deduplication:
    level: "info"
    enabled_fields:
      - message_id
      - hash
      - is_duplicate
      - processing_time_ms
      - cache_hit

# Cleanup и maintenance
maintenance:
  enabled: true
  cleanup_interval_seconds: 3600
  log_stats_interval_seconds: 300
```

---

## Enrichment Service Config

### config/config.enrichment.yaml
```yaml
# Enrichment Service Specific Configuration

service:
  name: "enrichment-service"
  version: "1.0.0"

enrichment:
  # Правила обогащения
  rules:
    # Правило 1: Обогащение профилем пользователя через API
    - id: "enrich-user-profile"
      name: "Enrich with user profile"
      enabled: true
      priority: 1
      field_to_enrich: "user_id"
      source_type: "api"
      source_config:
        url: "https://user-service/api/users/{user_id}"
        method: "GET"
        timeout_ms: 5000
        retry_count: 3
        headers:
          Authorization: "Bearer ${USER_SERVICE_TOKEN}"
          Content-Type: "application/json"
      
      transformation_rules:
        - source_path: "name"
          target_field: "user_profile.name"
          transform: "identity"
        - source_path: "account_age"
          target_field: "user_profile.account_age_days"
          transform: "identity"
        - source_path: "ltv"
          target_field: "user_profile.lifetime_value"
          transform: "identity"
      
      cache_ttl_seconds: 1800        # 30 minutes
      error_handling: "skip_field"   # skip_field, skip_rule, fail
      fallback_value: null
    
    # Правило 2: Обогащение геолокацией через API
    - id: "enrich-geolocation"
      name: "Enrich with geolocation"
      enabled: true
      priority: 2
      field_to_enrich: "country"
      source_type: "api"
      source_config:
        url: "https://geo-api/lookup?country={country}"
        method: "GET"
        timeout_ms: 3000
        retry_count: 2
      
      transformation_rules:
        - source_path: "city"
          target_field: "geo_data.city"
        - source_path: "region"
          target_field: "geo_data.region"
        - source_path: "timezone"
          target_field: "geo_data.timezone"
        - source_path: "lat"
          target_field: "geo_data.latitude"
        - source_path: "lng"
          target_field: "geo_data.longitude"
      
      cache_ttl_seconds: 3600
      error_handling: "skip_field"
    
    # Правило 3: Обогащение из MongoDB
    - id: "enrich-user-history"
      name: "Enrich with purchase history"
      enabled: true
      priority: 3
      field_to_enrich: "user_id"
      source_type: "database"
      source_config:
        database: "mongodb"
        collection: "user_history"
        query:
          user_id: "{user_id}"
        projection:
          total_purchases: 1
          avg_purchase_amount: 1
          last_purchase_date: 1
        limit: 1
      
      transformation_rules:
        - source_path: "total_purchases"
          target_field: "user_history.total_purchases"
      
      cache_ttl_seconds: 7200
      error_handling: "skip_field"
    
    # Правило 4: Обогащение из Cache (Redis)
    - id: "enrich-risk-score"
      name: "Enrich with risk score"
      enabled: true
      priority: 4
      field_to_enrich: "user_id"
      source_type: "cache"
      source_config:
        key_pattern: "risk_score:{user_id}"
        cache_type: "redis"
      
      transformation_rules:
        - source_path: "."
          target_field: "risk_score"
      
      error_handling: "skip_field"
  
  # Конфигурация кеша обогащения
  cache:
    enabled: true
    type: "redis"
    key_prefix: "enrich"
    default_ttl_seconds: 3600
    
    # Статистика
    track_statistics: true
    stats_interval_seconds: 60
  
  # Трансформация данных
  transformations:
    enabled: true
    
    # Поддерживаемые функции трансформации
    functions:
      identity:
        description: "Return value as-is"
      upper:
        description: "Convert to uppercase"
      lower:
        description: "Convert to lowercase"
      to_int:
        description: "Convert to integer"
      to_float:
        description: "Convert to float"
      json_parse:
        description: "Parse JSON string"
      concat:
        description: "Concatenate strings"
  
  # Обработка ошибок
  error_handling:
    # Если одно правило ошибется, не блокировать pipeline
    block_on_error: false
    
    # Dead Letter Queue для ошибочных обогащений
    dlq_enabled: true
    dlq_queue_name: "enrichment_errors"
    
    # Retry политика
    retry:
      max_attempts: 3
      backoff_type: "exponential"
      initial_delay_ms: 100
      max_delay_ms: 5000
  
  # Hot reload конфигурация
  hot_reload:
    enabled: true
    method: "event-driven"  # sighup, polling, event-driven
    polling_interval_seconds: 60

# Внешние сервисы / API
external_services:
  user_service:
    base_url: "${USER_SERVICE_URL:http://localhost:8081}"
    timeout_ms: 5000
    retry_count: 3
  
  geo_service:
    base_url: "${GEO_SERVICE_URL:http://localhost:8082}"
    timeout_ms: 3000
    retry_count: 2

# Метрики для обогащения
metrics:
  enrichment:
    enabled: true
    prefix: "enrichment"
    
    counters:
      - name: "messages_total"
      - name: "enriched"
      - name: "partial_enriched"
      - name: "errors"
      - name: "cache_hits"
      - name: "cache_misses"
      - name: "api_calls"
      - name: "api_errors"
    
    gauges:
      - name: "active_rules"
      - name: "cache_size"
      - name: "cache_hit_rate"
    
    histograms:
      - name: "processing_duration_ms"
        buckets: [1, 5, 10, 50, 100, 200, 500, 1000]
      - name: "api_response_time_ms"
        buckets: [10, 50, 100, 500, 1000, 2000]
      - name: "cache_latency_ms"
        buckets: [1, 5, 10, 20, 50]

# Логирование для обогащения
logging:
  enrichment:
    level: "info"
    enabled_fields:
      - message_id
      - rule_id
      - source_type
      - enriched_field
      - success
      - error
      - cache_hit
      - processing_time_ms
```

---

## Management Service Config

### config/config.management.yaml
```yaml
# Management Service Specific Configuration

service:
  name: "management-service"
  version: "1.0.0"
  
# REST API конфигурация
api:
  version: "v1"
  base_path: "/api/v1"
  
  endpoints:
    # Filtering rules endpoints
    filtering_rules:
      path: "/rules/filtering"
      methods: [GET, POST, PUT, DELETE]
    
    # Deduplication config endpoints
    deduplication_config:
      path: "/config/deduplication"
      methods: [GET, PUT]
    
    # Enrichment rules endpoints
    enrichment_rules:
      path: "/rules/enrichment"
      methods: [GET, POST, PUT, DELETE]
    
    # Health check
    health:
      path: "/health"
      methods: [GET]
    
    # Statistics
    stats:
      path: "/stats"
      methods: [GET]
    
    # Metrics
    metrics:
      path: "/metrics"
      methods: [GET]

# Аутентификация и авторизация
auth:
  enabled: true
  type: "jwt"  # jwt, api_key, oauth2
  
  jwt:
    secret_key: "${JWT_SECRET_KEY:your-secret-key}"
    algorithm: "HS256"
    expiration_hours: 24
  
  api_key:
    enabled: false
    header_name: "X-API-Key"
  
  # Roles и permissions
  rbac:
    enabled: true
    roles:
      - name: "admin"
        permissions:
          - "rules:create"
          - "rules:read"
          - "rules:update"
          - "rules:delete"
          - "config:update"
          - "stats:read"
      
      - name: "operator"
        permissions:
          - "rules:read"
          - "rules:update"
          - "stats:read"
      
      - name: "viewer"
        permissions:
          - "rules:read"
          - "stats:read"

# Валидация правил
validation:
  filtering:
    enabled: true
    rules:
      - field_required: true
      - operator_in: ["eq", "contains", "regex", "gt", "lt", "in", "range"]
      - value_required: true
      - value_type_validation: true
  
  enrichment:
    enabled: true
    rules:
      - field_to_enrich_required: true
      - source_type_in: ["api", "database", "cache", "file"]
      - source_config_required: true
  
  deduplication:
    enabled: true
    rules:
      - window_duration_positive: true
      - fields_not_empty: true

# Уведомления об изменениях (notify services)
notifications:
  enabled: true
  
  # Методы отправки уведомлений
  methods:
    - type: "rabbitmq"
      enabled: true
      exchange: "config.updates"
      routing_key: "rules.updated"
    
    - type: "http_webhook"
      enabled: false
      url: "${WEBHOOK_URL}"
      timeout_ms: 5000
      retry_count: 3
  
  # Какие события отправлять
  events:
    - filtering_rule_created
    - filtering_rule_updated
    - filtering_rule_deleted
    - enrichment_rule_created
    - enrichment_rule_updated
    - enrichment_rule_deleted
    - deduplication_config_updated

# Версионирование и аудит
versioning:
  enabled: true
  track_changes: true
  
audit_log:
  enabled: true
  storage: "postgresql"  # postgresql или mongodb
  
  # Какие события логировать
  events:
    - action: "create"
      resource: "filtering_rule"
    - action: "update"
      resource: "filtering_rule"
    - action: "delete"
      resource: "filtering_rule"
    - action: "update"
      resource: "deduplication_config"
    - action: "create"
      resource: "enrichment_rule"
    - action: "update"
      resource: "enrichment_rule"
    - action: "delete"
      resource: "enrichment_rule"

# Rate limiting
rate_limiting:
  enabled: true
  
  limits:
    # Для создания правил
    create_rule:
      requests: 100
      window_seconds: 3600
    
    # Для чтения
    read:
      requests: 10000
      window_seconds: 60
    
    # Для update
    update_rule:
      requests: 500
      window_seconds: 3600

# Caching for API responses
caching:
  enabled: true
  ttl_seconds: 300
  
  cache_endpoints:
    - path: "/rules/filtering"
      method: "GET"
      ttl: 300
    
    - path: "/rules/enrichment"
      method: "GET"
      ttl: 300
    
    - path: "/health"
      method: "GET"
      ttl: 30

# Метрики для Management Service
metrics:
  management:
    enabled: true
    prefix: "management"
    
    counters:
      - name: "api_requests_total"
      - name: "api_requests_success"
      - name: "api_requests_error"
      - name: "rule_created_total"
      - name: "rule_updated_total"
      - name: "rule_deleted_total"
    
    gauges:
      - name: "total_filtering_rules"
      - name: "total_enrichment_rules"
    
    histograms:
      - name: "api_response_time_ms"
        buckets: [1, 5, 10, 50, 100, 200, 500, 1000]

# Логирование для Management API
logging:
  management:
    level: "info"
    enabled_fields:
      - method
      - path
      - status_code
      - user_id
      - response_time_ms
      - resource_id
      - action
```

---

## Environment Variables

### .env.example
```bash
# =============================================
# ENVIRONMENT SETUP
# =============================================
ENVIRONMENT=development
LOG_LEVEL=info

# =============================================
# RABBITMQ CONFIGURATION
# =============================================
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest

# =============================================
# POSTGRESQL CONFIGURATION
# =============================================
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=admin
POSTGRES_PASSWORD=password
POSTGRES_DB=filtering

# =============================================
# REDIS CONFIGURATION
# =============================================
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# =============================================
# MONGODB CONFIGURATION
# =============================================
MONGODB_HOST=localhost
MONGODB_PORT=27017
MONGODB_USER=admin
MONGODB_PASSWORD=password
MONGODB_URI=mongodb://admin:password@localhost:27017
MONGODB_DB=enrichment

# =============================================
# SERVICE PORTS
# =============================================
FILTERING_SERVICE_PORT=8081
DEDUP_SERVICE_PORT=8082
ENRICHMENT_SERVICE_PORT=8083
MANAGEMENT_SERVICE_PORT=8084

# =============================================
# AUTHENTICATION
# =============================================
JWT_SECRET_KEY=your-secret-key-here
API_KEY=your-api-key-here

# =============================================
# EXTERNAL SERVICES
# =============================================
USER_SERVICE_URL=http://localhost:8081
USER_SERVICE_TOKEN=your-token
GEO_SERVICE_URL=http://localhost:8082

# =============================================
# MONITORING & TRACING
# =============================================
PROMETHEUS_PORT=9090
JAEGER_ENDPOINT=http://localhost:14268/api/traces
GRAFANA_PORT=3000

# =============================================
# DATABASE MIGRATIONS
# =============================================
RUN_MIGRATIONS=true
MIGRATION_PATH=./migrations

# =============================================
# HOT RELOAD
# =============================================
HOT_RELOAD_ENABLED=true
CONFIG_WATCH_INTERVAL=60
CONFIG_FILE_PATH=./config/config.base.yaml
```

---

## Примеры использования конфигурации

### Для разработки
```bash
# Загрузить config.base.yaml + config.dev.yaml
export ENVIRONMENT=development
go run cmd/filtering-service/main.go
```

### Для staging
```bash
# Загрузить config.base.yaml + config.staging.yaml
export ENVIRONMENT=staging
docker-compose -f docker-compose.staging.yml up
```

### Для production
```bash
# Загрузить config.base.yaml + config.prod.yaml
export ENVIRONMENT=production
docker-compose -f docker-compose.prod.yml up -d
```

---

## Hot Reload правил

### Способ 1: SIGHUP сигнал
```bash
# Пересылаем SIGHUP сигнал процессу
kill -SIGHUP <PID>

# Сервис перечитает конфигурацию
# Логи покажут: "Configuration reloaded successfully"
```

### Способ 2: REST API (Management Service)
```bash
# Создать новое правило фильтрации
curl -X POST http://localhost:8084/api/v1/rules/filtering \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New rule",
    "field": "status",
    "operator": "eq",
    "value": "active"
  }'

# Правило будет применено к новым сообщениям в течение минуты
```

### Способ 3: Polling (Автоматический)
```yaml
# config.base.yaml
filtering:
  hot_reload:
    enabled: true
    method: "polling"
    polling_interval_seconds: 60
```

---

