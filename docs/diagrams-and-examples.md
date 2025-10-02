# 🔄 Диаграммы взаимодействия и примеры данных

## Оглавление
1. [Message Flow Diagram](#message-flow-diagram)
2. [Hot Reload Scenarios](#hot-reload-scenarios)
3. [Error Handling Flow](#error-handling-flow)
4. [Примеры данных](#примеры-данных)
5. [Sequence Diagrams](#sequence-diagrams)

---

## Message Flow Diagram

### Полный путь сообщения через систему

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            INPUT MESSAGE                                     │
│  {                                                                           │
│    "id": "msg-12345",                                                       │
│    "timestamp": "2025-12-14T14:55:00Z",                                     │
│    "source": "api-gateway",                                                 │
│    "user_id": "user-789",                                                   │
│    "event_type": "purchase",                                                │
│    "amount": 99.99,                                                         │
│    "status": "active",                                                      │
│    "email": "user@example.com",                                             │
│    "country": "US"                                                          │
│  }                                                                           │
└──────────────────────────┬──────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │  Kafka: input_events topic        │
        │  (durable, consumer group)        │
        └──────────────────┬───────────────┘
                           │
                           ▼
        ╔════════════════════════════════════════════╗
        ║      FILTERING SERVICE (Port 8081)         ║
        ╠════════════════════════════════════════════╣
        ║                                            ║
        ║  Rules from PostgreSQL:                    ║
        ║  1. status = "active" (PASS)               ║
        ║  2. amount > 10 (PASS)                     ║
        ║  3. email matches regex (PASS)             ║
        ║                                            ║
        ║  Result: Message PASSED ✓                  ║
        ║                                            ║
        ║  Metrics: filtering_messages_passed++      ║
        ╚════════════════════════┬════════════════════╝
                                 │
         ┌───────────────────────▼──────────────────────┐
         │  MESSAGE WITH METADATA                       │
         │  {                                           │
         │    ...previous fields...,                    │
         │    "filters_applied": {                      │
         │      "rule_ids": ["rule-1", "rule-2", ...], │
         │      "passed_at": "2025-12-14T14:55:00.123Z"│
         │    }                                         │
         │  }                                           │
         └───────────────────────┬──────────────────────┘
                                 │
        ┌────────────────────────▼───────────────────┐
        │  Kafka: dedup_events topic                  │
        │  (durable, consumer group)                  │
        └────────────────────────┬───────────────────┘
                                 │
                                 ▼
        ╔════════════════════════════════════════════╗
        ║   DEDUPLICATION SERVICE (Port 8082)        ║
        ╠════════════════════════════════════════════╣
        ║                                            ║
        ║  Compute hash:                             ║
        ║  hash = md5(id+timestamp+source)           ║
        ║  hash = "a1b2c3d4e5f6..."                  ║
        ║                                            ║
        ║  Check Redis:                              ║
        ║  SET dedup:a1b2c3d4e5f6 1734268500 EX 3600 ║
        ║  Result: NX SET SUCCESS (unique)           ║
        ║                                            ║
        ║  Metrics: dedup_unique_messages++          ║
        ║  Metrics: dedup_cache_misses++             ║
        ╚════════════════════════┬════════════════════╝
                                 │
         ┌───────────────────────▼──────────────────────┐
         │  MESSAGE WITH METADATA                       │
         │  {                                           │
         │    ...previous fields...,                    │
         │    \"deduplication\": {                      │
         │      \"is_unique\": true,                    │
         │      \"hash\": \"a1b2c3d4e5f6...\",          │
         │      \"checked_at\": \"2025-12-14T...\"      │
         │    }                                         │
         │  }                                           │
         └───────────────────────┬──────────────────────┘
                                 │
        ┌────────────────────────▼──────────────────────┐
        │  Kafka: enrichment_events topic               │
        │  (durable, consumer group)                     │
        └────────────────────────┬──────────────────────┘
                                 │
                                 ▼
        ╔════════════════════════════════════════════╗
        ║    ENRICHMENT SERVICE (Port 8083)          ║
        ╠════════════════════════════════════════════╣
        ║                                            ║
        ║  Rule 1: Enrich with user profile (API)   ║
        ║  ├─ Cache check: MISS                      ║
        ║  ├─ API call: GET /users/user-789         ║
        ║  ├─ Response: {name, account_age, ltv}    ║
        ║  └─ Cache: enrich:profile:user-789 1800s  ║
        ║                                            ║
        ║  Rule 2: Enrich with geolocation (API)    ║
        ║  ├─ Cache check: HIT!                      ║
        ║  ├─ Use cached: {city, region, tz}        ║
        ║  └─ Metrics: cache_hit_rate++             ║
        ║                                            ║
        ║  Rule 3: Enrich with history (MongoDB)    ║
        ║  ├─ Query: db.user_history.findOne(...)   ║
        ║  └─ Result: {total_purchases, ...}        ║
        ║                                            ║
        ║  Metrics: enrichment_processed++           ║
        ║  Metrics: enrichment_cache_hit_rate = 0.67 ║
        ║  Metrics: api_calls_total += 2             ║
        ╚════════════════════════┬════════════════════╝
                                 │
         ┌───────────────────────▼──────────────────────┐
         │  FINAL MESSAGE                               │
         │  {                                           │
         │    \"id\": \"msg-12345\",                    │
         │    \"timestamp\": \"2025-12-14T14:55:00Z\",  │
         │    \"source\": \"api-gateway\",              │
         │    \"user_id\": \"user-789\",                │
         │    \"event_type\": \"purchase\",             │
         │    \"amount\": 99.99,                        │
         │    \"status\": \"active\",                   │
         │    \"email\": \"user@example.com\",          │
         │    \"country\": \"US\",                      │
         │    \"filters_applied\": {...},              │
         │    \"deduplication\": {...},                │
         │    \"enrichment\": {                         │
         │      \"user_profile\": {                     │
         │        \"name\": \"John Doe\",              │
         │        \"account_age_days\": 365,           │
         │        \"lifetime_value\": 5000.00          │
         │      },                                     │
         │      \"geo_data\": {                        │
         │        \"city\": \"New York\",              │
         │        \"region\": \"NY\",                  │
         │        \"timezone\": \"America/New_York\"   │
         │      },                                     │
         │      \"user_history\": {                    │
         │        \"total_purchases\": 42              │
         │      },                                     │
         │      \"rules_applied\": [\"rule-1\", ...], │
         │      \"enriched_at\": \"2025-12-14T...\"    │
         │    }                                        │
         │  }                                          │
         └───────────────────────┬──────────────────────┘
                                 │
        ┌────────────────────────▼──────────────────────┐
        │  Kafka: processed_events topic                 │
        │  (durable, ready for downstream consumers)     │
        └────────────────────────┬──────────────────────┘
                                 │
                                 ▼
                    ┌─────────────────────────┐
                    │   OUTPUT MESSAGE        │
                    │   (Ready for use)       │
                    └─────────────────────────┘
```

---

## Hot Reload Scenarios

### Scenario 1: Admin добавляет новое правило фильтрации

```
Timeline:
T=0s    Admin отправляет: POST /api/v1/rules/filtering
            {
              "name": "New rule",
              "field": "status",
              "operator": "eq",
              "value": "premium"
            }

T=0.1s  Management Service:
        1. Валидирует правило
        2. Сохраняет в PostgreSQL
        3. Создает version entry
        4. Публикует в Kafka: config_updates topic
        5. Возвращает 201 Created

T=0.2s  Filtering Service:
        (Method 1: Event-driven)
        ├─ Слушает Kafka: config_updates topic
        ├─ Получает: filtering.rule_updated
        ├─ Вызывает: ReloadRules()
        ├─ Блокирует: fs.rulesMu (write lock)
        ├─ Загружает из PostgreSQL
        ├─ Компилирует regex patterns
        ├─ Освобождает: fs.rulesMu
        └─ Логирует: "Rules reloaded successfully"

        (Method 2: Polling)
        ├─ Background goroutine проверяет каждые 60s
        ├─ Видит updated_at новее чем last_check
        ├─ Вызывает: ReloadRules()
        └─ Логирует: "Rules reloaded from polling"

T=0.3s  Новые сообщения используют новое правило
        ├─ Старые правила остаются в памяти
        ├─ Нет рестарта сервиса
        └─ Нет потери сообщений

Result: Новое правило активно через <1 секунду (event-driven)
        или максимум через 60 секунд (polling)
```

### Scenario 2: Admin обновляет окно дедубликации

```
T=0s    Admin: PUT /api/v1/config/deduplication
            { "window_seconds": 7200 }  // изменили с 3600

T=0.1s  Management Service:
        1. Валидирует конфиг
        2. Сохраняет в PostgreSQL
        3. Публикует в Kafka: config_updates topic

T=0.2s  Deduplication Service:
        1. Получает событие
        2. ds.window = 7200 seconds
        3. Логирует обновление
        
Behavior:
- Сообщения с хешами, добавленными 60 минут назад:
  ├─ Старый window: TTL истекло → treated as unique
  ├─ Новый window: остаток TTL все еще действителен
  └─ Redis не перезаписывает ключи

- Новые сообщения:
  └─ Используют новый window (7200s)

Result: Плавное обновление без потери данных
```

### Scenario 3: Admin изменяет API endpoint для обогащения

```
T=0s    Admin: PUT /api/v1/rules/enrichment/rule-1
            {
              "source_config": {
                "url": "https://new-api.example.com/users/{user_id}"
              }
            }

T=0.1s  Management Service:
        1. Сохраняет новый URL в MongoDB
        2. Публикует: enrichment.rule_updated

T=0.2s  Enrichment Service:
        1. Получает событие
        2. Перезагружает правила из MongoDB
        3. Кеш остается (будет выполняться с новым URL)

T=0.3s  Новые запросы:
        ├─ Идут на новый API endpoint
        ├─ Результаты кешируются (если кешь истекает)
        └─ Старые результаты в кеше используются (с TTL)

Result: Миграция на новый API без перезагрузки
```

---

## Error Handling Flow

### Scenario 1: Сообщение не проходит фильтрацию

```
Input Message:
{
  "id": "msg-wrong",
  "status": "inactive",   // ← не совпадает с фильтром
  "amount": 50
}

Filtering Service Processing:
1. Load rules (status = "active")
2. Extract field: msg["status"] = "inactive"
3. Apply operator: "inactive" == "active" → FALSE
4. Decision: FILTER OUT
5. Action:
   ├─ Increment: filtering_messages_filtered++
   ├─ Log: debug level (no error)
   └─ Message dropped (no further processing)

Result:
├─ Message не попадает в dedup_events queue
├─ Не обогащается
├─ Не добавляется в processed_events
└─ Это нормальное поведение (не ошибка)
```

### Scenario 2: Redis недоступен при дедубликации

```
Deduplication Service:
1. Compute hash: "a1b2c3d4..."
2. Try to SET in Redis → TIMEOUT (connection refused)
3. Error handling:
   ├─ config: on_redis_error = "allow" (или "deny")
   ├─ Strategy: Retry 3 times with exponential backoff
   │   ├─ Attempt 1 (after 100ms): FAIL
   │   ├─ Attempt 2 (after 200ms): FAIL
   │   └─ Attempt 3 (after 400ms): FAIL
   └─ Max retries reached:
       ├─ If allow: Send message to enrichment (assume unique)
       ├─ If deny: Send to DLQ (manual review)
       └─ Log: ERROR with details

DLQ Message:
{
  "original_message": { ... },
  "error": "redis connection timeout",
  "service": "dedup-service",
  "timestamp": "2025-12-14T14:55:00Z",
  "retry_count": 3
}

Recovery:
├─ Manual: Review DLQ, fix Redis, reprocess
├─ Automatic: Retry handler polls DLQ every 5 minutes
└─ Monitoring: Alert on DLQ depth > 100
```

### Scenario 3: External API timeout при обогащении

```
Enrichment Service:
Rule: Enrich with user profile via API

1. Check cache: MISS
2. Make API call: GET https://user-api/users/user-789
3. Timeout after 5 seconds (config: timeout_ms=5000)
4. Retry logic:
   ├─ Attempt 1 (immediate): TIMEOUT
   ├─ Attempt 2 (after 100ms): TIMEOUT
   └─ Attempt 3 (after 200ms): SUCCESS (API recovered)
5. Add enrichment: { user_profile: {...} }
6. Cache result (TTL: 1800s)

If all retries failed:
├─ config: error_handling = "skip_field"
├─ Result: Message passes without user_profile field
├─ Log: WARN level
├─ Metrics: enrichment_api_errors++
└─ Pipeline continues (not blocked)

If config = "fail":
├─ Send to enrichment_events_dlq
├─ Manual intervention needed
└─ Pipeline blocked for this message
```

### Scenario 4: Database constraint violation при создании правила

```
Admin API Request:
POST /api/v1/rules/filtering
{
  "name": "Duplicate rule",  // ← Имя уже существует
  "field": "status",
  ...
}

Management Service:
1. Validate rule structure ✓
2. Check uniqueness of name
3. Database returns: UNIQUE constraint violation
4. Handle error:
   ├─ Rollback transaction
   ├─ Return 400 Bad Request
   └─ Response:
       {
         "error": "Rule with this name already exists",
         "error_code": "DUPLICATE_NAME",
         "timestamp": "2025-12-14T14:55:00Z"
       }

Result:
├─ No partial updates
├─ Audit log not created
├─ Services not notified
└─ Admin can retry with different name
```

---

## Примеры данных

### Input Message Format

```json
{
  "id": "msg-uuid-12345",
  "timestamp": "2025-12-14T14:55:00.000Z",
  "source": "api-gateway",
  "correlation_id": "corr-uuid-98765",
  "trace_id": "trace-uuid-54321",
  
  "user_id": "user-789",
  "email": "john.doe@example.com",
  "event_type": "purchase",
  "amount": 99.99,
  "currency": "USD",
  "status": "active",
  "country": "US",
  "subscription_tier": "premium",
  
  "metadata": {
    "device": "mobile",
    "app_version": "2.1.0",
    "ip_address": "192.168.1.1"
  },
  
  "payload": {
    "product_id": "prod-123",
    "product_name": "Premium Subscription",
    "quantity": 1,
    "payment_method": "credit_card"
  }
}
```

### After Filtering

```json
{
  "id": "msg-uuid-12345",
  "timestamp": "2025-12-14T14:55:00.000Z",
  "source": "api-gateway",
  "correlation_id": "corr-uuid-98765",
  "trace_id": "trace-uuid-54321",
  
  "user_id": "user-789",
  "email": "john.doe@example.com",
  "event_type": "purchase",
  "amount": 99.99,
  "currency": "USD",
  "status": "active",
  "country": "US",
  "subscription_tier": "premium",
  
  "metadata": { ... },
  "payload": { ... },
  
  "filters_applied": {
    "rule_ids": ["filter-status-active", "filter-event-purchase", "filter-valid-email", "filter-amount-min"],
    "passed_at": "2025-12-14T14:55:00.123Z",
    "processing_time_ms": 2.5
  }
}
```

### After Deduplication

```json
{
  "id": "msg-uuid-12345",
  ...previous fields...,
  
  "filters_applied": { ... },
  
  "deduplication": {
    "is_unique": true,
    "hash": "a1b2c3d4e5f6g7h8i9j0",
    "checked_at": "2025-12-14T14:55:00.456Z",
    "processing_time_ms": 1.2
  }
}
```

### After Enrichment

```json
{
  "id": "msg-uuid-12345",
  ...previous fields...,
  
  "filters_applied": { ... },
  "deduplication": { ... },
  
  "enrichment": {
    "user_profile": {
      "name": "John Doe",
      "account_age_days": 365,
      "subscription_tier": "premium",
      "lifetime_value": 5000.00,
      "account_created_at": "2024-12-14T00:00:00Z"
    },
    
    "geo_data": {
      "city": "New York",
      "region": "NY",
      "country": "United States",
      "timezone": "America/New_York",
      "latitude": 40.7128,
      "longitude": -74.0060,
      "is_vpn": false
    },
    
    "risk_assessment": {
      "fraud_score": 0.15,
      "risk_level": "low",
      "flags": []
    },
    
    "purchase_history": {
      "total_purchases": 42,
      "avg_purchase_value": 119.05,
      "last_purchase_date": "2025-12-10T00:00:00Z",
      "repeat_customer": true
    },
    
    "rules_applied": [
      "enrich-user-profile",
      "enrich-geolocation",
      "enrich-purchase-history"
    ],
    
    "enriched_at": "2025-12-14T14:55:00.789Z",
    "processing_time_ms": 45.3,
    "cache_hits": 2,
    "cache_misses": 1
  }
}
```

### Error Message (DLQ)

```json
{
  "message_id": "msg-uuid-error-12345",
  "timestamp": "2025-12-14T14:55:00Z",
  
  "error_details": {
    "error_type": "redis_timeout",
    "service": "dedup-service",
    "error_message": "Redis connection timeout after 10s",
    "stack_trace": "...",
    "retry_count": 3
  },
  
  "original_message": {
    "id": "msg-uuid-12345",
    ...
  },
  
  "routing_info": {
    "source_queue": "dedup_events",
    "destination_queue": "dedup_events_dlq",
    "dead_letter_reason": "processing_error"
  }
}
```

---

## Sequence Diagrams

### Success Path (Full Pipeline)

```
Admin            Management    Filtering    Broker      Dedup        Enrich       Output
  │                  │             │          │           │             │            │
  │─ POST rule ────→ │             │          │           │             │            │
  │                  │ validate    │          │           │             │            │
  │                  │ save to DB  │          │           │             │            │
  │                  │ publish     │          │           │             │            │
  │                  │────────────────────→ (config.updates)            │            │
  │                  │             │          │           │             │            │
  │                  │─────────────────────────────────────────────────────→ reload  │
  │                  │             │          │           │             │            │
  │  ← 201 Created ──│             │          │           │             │            │
  │                  │             │          │           │             │            │
  │                  │             │ ← message ← (input_events queue)   │            │
  │                  │             │ filter   │           │             │            │
  │                  │             │───────────────────→ (dedup_events) │            │
  │                  │             │          │           │             │            │
  │                  │             │          │           │ → dedup     │            │
  │                  │             │          │           │ → cache     │            │
  │                  │             │          │           │───────────────→ enrich  │
  │                  │             │          │           │             │           │
  │                  │             │          │           │             │ → API     │
  │                  │             │          │           │             │ → cache   │
  │                  │             │          │           │             │───────────→ queue
  │                  │             │          │           │             │            │
  │                  │             │          │           │             │            │ ← output
  │                  │             │          │           │             │            │ ready
```

### Error Path (Retry + DLQ)

```
Message         Filtering    Broker       Dedup        (Redis Error)
   │                │          │            │              │
   │─────────────────→ pass    │            │              │
   │                │──────────────────→   │              │
   │                │          │            │ SET attempt  │
   │                │          │            │─────────→ TIMEOUT
   │                │          │            │              │
   │                │          │            │ ← RETRY 1 (100ms)
   │                │          │            │─────────→ TIMEOUT
   │                │          │            │              │
   │                │          │            │ ← RETRY 2 (200ms)
   │                │          │            │─────────→ TIMEOUT
   │                │          │            │              │
   │                │          │            │ ← RETRY 3 (400ms)
   │                │          │            │─────────→ TIMEOUT
   │                │          │            │              │
   │                │          │            │ Max retries → DLQ
   │                │          │            │──────────────────→ (dedup_events_dlq)
   │                │          │            │                    │
   │                │          │            │                    │ [Manual Review]
   │                │          │            │                    │
   │                │          │            │                    │ [Fix Redis]
   │                │          │            │                    │
   │                │          │            │  ← [Reprocess]
   │                │          │            │─────────→ SUCCESS
```

---

