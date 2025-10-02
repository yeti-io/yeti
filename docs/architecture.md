# 🏗️ Архитектура системы обработки данных

## Оглавление
1. [Общее описание](#общее-описание)
2. [Диаграмма системы](#диаграмма-системы)
3. [Компоненты системы](#компоненты-системы)
4. [Взаимодействие между сервисами](#взаимодействие-между-сервисами)
5. [Внутренние контракты](#внутренние-контракты)

---

## Общее описание

Система представляет собой распределенный pipeline обработки данных с независимыми микросервисами, каждый из которых специализируется на одном этапе обработки. Архитектура позволяет:

- ✅ Масштабировать каждый компонент независимо
- ✅ Обновлять правила обработки без перезагрузки сервисов
- ✅ Гарантировать надежность каждого этапа обработки
- ✅ Мониторить и отслеживать процесс обработки
- ✅ Легко добавлять новые этапы в будущем

---

## Диаграмма системы

```
┌──────────────────────────────────────────────────────────────────┐
│                         INPUT MESSAGE                             │
│                      (Incoming Data Stream)                       │
└──────────────────────┬───────────────────────────────────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │   Message Broker - INPUT     │
        │   (Kafka)                     │
        │   Topic: input_events        │
        └──────────────────┬───────────┘
                           │
        ┌──────────────────▼───────────────────┐
        │   FILTERING SERVICE                   │
        ├───────────────────────────────────────┤
        │ • Читает правила фильтрации           │
        │ • Применяет фильтры к сообщениям     │
        │ • Отправляет пройденные сообщения   │
        │                                       │
        │ Database: PostgreSQL                  │
        │ Rules Cache: In-Memory (RWMutex)     │
        └──────────────────┬────────────────────┘
                           │ (passed messages)
        ┌──────────────────▼───────────────────┐
        │   Message Broker - DEDUP INPUT       │
        │   Topic: dedup_events                 │
        └──────────────────┬────────────────────┘
                           │
        ┌──────────────────▼───────────────────┐
        │   DEDUPLICATION SERVICE               │
        ├───────────────────────────────────────┤
        │ • Проверяет уникальность сообщения   │
        │ • Хранит хеши в Redis с TTL          │
        │ • Отправляет уникальные сообщения   │
        │                                       │
        │ Database: Redis (with TTL)           │
        │ Window: Configurable (default 1h)   │
        └──────────────────┬────────────────────┘
                           │ (unique messages)
        ┌──────────────────▼───────────────────┐
        │   Message Broker - ENRICHMENT INPUT  │
        │   Topic: enrichment_events            │
        └──────────────────┬────────────────────┘
                           │
        ┌──────────────────▼───────────────────┐
        │   ENRICHMENT SERVICE                  │
        ├───────────────────────────────────────┤
        │ • Применяет правила обогащения       │
        │ • Загружает данные из внешних API   │
        │ • Добавляет контекст к сообщениям   │
        │                                       │
        │ Database: MongoDB                     │
        │ Cache: Redis (для частых запросов)  │
        └──────────────────┬────────────────────┘
                           │ (enriched messages)
        ┌──────────────────▼───────────────────┐
        │   Message Broker - OUTPUT             │
        │   Topic: processed_events             │
        └──────────────────┬────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────┐
        │   OUTPUT MESSAGE             │
        │   (Ready for Consumption)    │
        └──────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     MANAGEMENT SERVICE (REST API)                │
├─────────────────────────────────────────────────────────────────┤
│ Endpoints:                                                        │
│ • GET/POST/PUT/DELETE  /api/v1/rules/filtering                  │
│ • GET/POST/PUT/DELETE  /api/v1/rules/deduplication              │
│ • GET/POST/PUT/DELETE  /api/v1/rules/enrichment                 │
│ • GET                  /api/v1/health                            │
│ • GET                  /api/v1/stats                             │
│ • GET                  /api/v1/metrics                           │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│              MONITORING & LOGGING                                │
├─────────────────────────────────────────────────────────────────┤
│ • Prometheus:  :8080/metrics (от каждого сервиса)              │
│ • Grafana:     :3000 (визуализация метрик)                     │
│ • ELK Stack:   :5601 (логирование - опционально)               │
│ • Jaeger:      :16686 (distributed tracing - опционально)      │
└─────────────────────────────────────────────────────────────────┘
```

---

## Компоненты системы

### 1. **Filtering Service** 🔍

#### Назначение
Первый этап pipeline - фильтрация сообщений по заданным правилам. Отсеивает нежелательные сообщения до того, как они будут обработаны на следующих этапах.

#### Ответственность
- Загрузка и кеширование правил фильтрации
- Применение правил к входящим сообщениям
- Hot reload правил без перезагрузки сервиса
- Метрики обработки (пройденные/отфильтрованные сообщения)

#### Типы правил фильтрации
```
1. Exact Match (eq)         - точное совпадение значения
2. Contains (contains)      - содержит подстроку
3. Regex (regex)            - соответствие регулярному выражению
4. Greater Than (gt)        - больше значения (числовое)
5. Less Than (lt)           - меньше значения (числовое)
6. In List (in)             - значение из списка
7. Range (range)            - значение в диапазоне [min, max]
```

#### Входные данные
```json
{
  "id": "msg-12345",
  "timestamp": "2025-12-14T14:55:00Z",
  "source": "api-gateway",
  "user_id": "user-789",
  "event_type": "purchase",
  "amount": 99.99,
  "status": "active",
  "email": "user@example.com",
  "payload": {...}
}
```

#### Выходные данные
```json
{
  "id": "msg-12345",
  "timestamp": "2025-12-14T14:55:00Z",
  "source": "api-gateway",
  "user_id": "user-789",
  "event_type": "purchase",
  "amount": 99.99,
  "status": "active",
  "email": "user@example.com",
  "payload": {...},
  "filters_applied": {
    "rule_ids": ["rule-1", "rule-2"],
    "passed_at": "2025-12-14T14:55:00.123Z"
  }
}
```

#### Database Schema (PostgreSQL)
```sql
-- Таблица правил фильтрации
CREATE TABLE filtering_rules (
  id UUID PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  field VARCHAR(255) NOT NULL,
  operator VARCHAR(50) NOT NULL,
  value JSONB NOT NULL,
  enabled BOOLEAN DEFAULT true,
  priority INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  version INTEGER DEFAULT 1
);

-- Индексы
CREATE INDEX idx_filtering_rules_enabled ON filtering_rules(enabled);
CREATE INDEX idx_filtering_rules_priority ON filtering_rules(priority DESC);

-- Таблица логов применения правил (опционально)
CREATE TABLE filtering_logs (
  id UUID PRIMARY KEY,
  rule_id UUID REFERENCES filtering_rules(id),
  message_id VARCHAR(255),
  matched BOOLEAN,
  execution_time_ms INTEGER,
  created_at TIMESTAMP DEFAULT NOW()
);
```

#### Кеширование
- **В памяти**: RWMutex защищенный map[string]*CompiledRule
- **Обновление**: На SIGHUP сигнал или через API
- **Компиляция Regex**: При загрузке правил
- **TTL**: Не требуется (перезагрузка по событию)

---

### 2. **Deduplication Service** 🔄

#### Назначение
Второй этап pipeline - обнаружение и фильтрация дубликатов. Гарантирует, что одно и то же сообщение не будет обработано несколько раз.

#### Ответственность
- Вычисление хеша сообщения
- Проверка наличия в Redis с временным окном
- Управление TTL для окна дедубликации
- Метрики дубликатов и уникальных сообщений

#### Алгоритм дедубликации
```
1. Извлечь ключевые поля из сообщения (id, timestamp, source)
2. Вычислить MD5 хеш: hash = md5(id + timestamp + source)
3. Попытаться установить в Redis: SET dedup:{hash} {timestamp} EX {window}
4. Если SET успешен -> сообщение уникально
5. Если SET вернул nil -> сообщение дубликат
```

#### Входные данные
```json
{
  "id": "msg-12345",
  "timestamp": "2025-12-14T14:55:00Z",
  "source": "api-gateway",
  // ... остальные поля
  "filters_applied": {
    "rule_ids": ["rule-1", "rule-2"],
    "passed_at": "2025-12-14T14:55:00.123Z"
  }
}
```

#### Выходные данные
```json
{
  "id": "msg-12345",
  "timestamp": "2025-12-14T14:55:00Z",
  "source": "api-gateway",
  // ... остальные поля
  "filters_applied": {
    "rule_ids": ["rule-1", "rule-2"],
    "passed_at": "2025-12-14T14:55:00.123Z"
  },
  "deduplication": {
    "is_unique": true,
    "hash": "abc123def456",
    "checked_at": "2025-12-14T14:55:00.456Z"
  }
}
```

#### Redis Schema
```
Key Pattern:    dedup:{hash}
Value:          timestamp (Unix)
TTL:            configurable (default 3600 seconds / 1 hour)

Example:
Key:    dedup:a1b2c3d4e5f6g7h8
Value:  1734268500
TTL:    3600 (expires automatically)
```

#### Конфигурация
```yaml
deduplication:
  window_seconds: 3600        # 1 час
  fields_for_hash:
    - id
    - timestamp
    - source
  hash_algorithm: md5         # или sha256
```

#### Обработка ошибок
- **Redis недоступен**: Отправить сообщение в DLQ, залогировать ошибку
- **Невалидные данные**: Пропустить дедубликацию, залогировать warning
- **Timeout Redis**: Retry 3 раза с exponential backoff

---

### 3. **Enrichment Service** 🎁

#### Назначение
Третий этап pipeline - обогащение сообщений дополнительной информацией из внешних источников или внутренних баз данных.

#### Ответственность
- Загрузка и управление правилами обогащения
- Вызов внешних API/сервисов
- Трансформация и добавление данных
- Кеширование результатов обогащения
- Hot reload правил

#### Типы источников обогащения
```
1. API Services       - вызовы HTTP API
2. Database Lookup    - запросы к MongoDB/PostgreSQL
3. Cache Lookup       - получение данных из Redis
4. File Lookup        - загрузка статических данных
5. ML Model           - вызовы ML сервисов
6. Aggregation        - агрегирование данных из разных источников
```

#### Входные данные
```json
{
  "id": "msg-12345",
  "timestamp": "2025-12-14T14:55:00Z",
  "source": "api-gateway",
  "user_id": "user-789",
  "email": "user@example.com",
  "event_type": "purchase",
  "amount": 99.99,
  "country": "US",
  "filters_applied": {...},
  "deduplication": {...}
}
```

#### Выходные данные
```json
{
  "id": "msg-12345",
  "timestamp": "2025-12-14T14:55:00Z",
  "source": "api-gateway",
  "user_id": "user-789",
  "email": "user@example.com",
  "event_type": "purchase",
  "amount": 99.99,
  "country": "US",
  "filters_applied": {...},
  "deduplication": {...},
  "enrichment": {
    "user_profile": {
      "name": "John Doe",
      "account_age_days": 365,
      "subscription_tier": "premium",
      "lifetime_value": 5000.00
    },
    "geo_data": {
      "city": "New York",
      "region": "NY",
      "timezone": "America/New_York",
      "latitude": 40.7128,
      "longitude": -74.0060
    },
    "risk_score": 0.15,
    "rules_applied": ["rule-1", "rule-3"],
    "enriched_at": "2025-12-14T14:55:00.789Z"
  }
}
```

#### Database Schema (MongoDB)
```javascript
// Collections: enrichment_rules
db.enrichment_rules.createCollection("enrichment_rules", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["_id", "name", "source_type", "enabled"],
      properties: {
        _id: { bsonType: "objectId" },
        name: { bsonType: "string", description: "Имя правила обогащения" },
        field_to_enrich: { bsonType: "string", description: "Поле в сообщении" },
        source_type: { 
          enum: ["api", "database", "cache", "file", "ml"],
          description: "Тип источника"
        },
        source_config: { bsonType: "object", description: "Конфигурация источника" },
        transformation_rules: { bsonType: "array", description: "Правила трансформации" },
        cache_ttl_seconds: { bsonType: "int", description: "TTL кеша" },
        enabled: { bsonType: "bool" },
        priority: { bsonType: "int", default: 0 },
        created_at: { bsonType: "date" },
        updated_at: { bsonType: "date" }
      }
    }
  }
});

// Collections: enrichment_cache
db.enrichment_cache.createCollection("enrichment_cache", {});
// TTL index
db.enrichment_cache.createIndex(
  { "created_at": 1 },
  { expireAfterSeconds: 3600 }
);
```

#### Кеширование
- **Redis**: Кеш результатов обогащения с TTL
- **Ключ**: `enrich:{rule_id}:{source_hash}`
- **Значение**: JSON результат обогащения
- **TTL**: Из конфига правила (default 3600 sec)

#### Обработка ошибок
```
Если обогащение от источника не удалось:
1. Log ошибку с details
2. Проверить fallback в конфиге
3. Если fallback есть - использовать его
4. Если fallback нет - пропустить обогащение этого поля
5. Отправить сообщение дальше (не блокировать pipeline)
```

---

### 4. **Management Service** 🎛️

#### Назначение
REST API для управления правилами всех сервисов. Позволяет администраторам изменять правила без изменения кода.

#### Ответственность
- CRUD операции над правилами фильтрации
- CRUD операции над правилами дедубликации
- CRUD операции над правилами обогащения
- Рассылка уведомлений об изменениях
- Validация правил перед сохранением
- Управление версиями правил
- Статистика и мониторинг

#### REST API Endpoints

**Filtering Rules**
```
GET     /api/v1/rules/filtering           - список всех правил
POST    /api/v1/rules/filtering           - создать новое правило
GET     /api/v1/rules/filtering/:id       - получить правило по ID
PUT     /api/v1/rules/filtering/:id       - обновить правило
DELETE  /api/v1/rules/filtering/:id       - удалить правило
PATCH   /api/v1/rules/filtering/:id/toggle - включить/отключить
```

**Deduplication Rules**
```
GET     /api/v1/rules/deduplication       - текущие параметры дедубликации
PUT     /api/v1/rules/deduplication       - обновить параметры
GET     /api/v1/rules/deduplication/stats - статистика дубликатов
```

**Enrichment Rules**
```
GET     /api/v1/rules/enrichment          - список всех правил
POST    /api/v1/rules/enrichment          - создать новое правило
GET     /api/v1/rules/enrichment/:id      - получить правило по ID
PUT     /api/v1/rules/enrichment/:id      - обновить правило
DELETE  /api/v1/rules/enrichment/:id      - удалить правило
PATCH   /api/v1/rules/enrichment/:id/toggle - включить/отключить
```

**System**
```
GET     /api/v1/health                    - проверка здоровья
GET     /api/v1/health/services           - статус всех сервисов
GET     /api/v1/stats                     - статистика обработки
GET     /api/v1/stats/pipeline            - статистика по этапам
GET     /api/v1/metrics                   - Prometheus метрики
POST    /api/v1/rules/reload-signal       - отправить SIGHUP сигнал
```

#### Request/Response Examples

**Create Filtering Rule**
```http
POST /api/v1/rules/filtering
Content-Type: application/json

{
  "name": "Filter by status",
  "field": "status",
  "operator": "eq",
  "value": "active",
  "priority": 1,
  "enabled": true
}

Response 201:
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Filter by status",
  "field": "status",
  "operator": "eq",
  "value": "active",
  "priority": 1,
  "enabled": true,
  "created_at": "2025-12-14T14:55:00Z",
  "updated_at": "2025-12-14T14:55:00Z",
  "version": 1
}
```

**Update Enrichment Rule**
```http
PUT /api/v1/rules/enrichment/550e8400-e29b-41d4-a716-446655440001
Content-Type: application/json

{
  "name": "Enrich with user profile",
  "source_type": "api",
  "source_config": {
    "url": "https://user-service/api/users/{user_id}",
    "timeout_ms": 5000,
    "retry_count": 3
  },
  "cache_ttl_seconds": 1800,
  "enabled": true
}

Response 200: {...обновленное правило...}
```

**Get Stats**
```http
GET /api/v1/stats

Response 200:
{
  "timestamp": "2025-12-14T14:55:00Z",
  "total_messages_processed": 1000000,
  "pipeline": {
    "filtering": {
      "input": 1000000,
      "passed": 850000,
      "filtered": 150000,
      "error_rate": 0.001
    },
    "deduplication": {
      "input": 850000,
      "unique": 800000,
      "duplicate": 50000,
      "error_rate": 0.0
    },
    "enrichment": {
      "input": 800000,
      "enriched": 790000,
      "partial_enrich": 10000,
      "error_rate": 0.002,
      "cache_hit_rate": 0.75
    }
  },
  "uptime_seconds": 86400
}
```

#### Database Schema (PostgreSQL)
```sql
-- Версионирование правил
CREATE TABLE rule_versions (
  id UUID PRIMARY KEY,
  rule_id UUID NOT NULL,
  rule_type VARCHAR(50) NOT NULL,
  rule_data JSONB NOT NULL,
  version INTEGER NOT NULL,
  changed_by VARCHAR(255),
  created_at TIMESTAMP DEFAULT NOW(),
  UNIQUE(rule_id, version)
);

-- Аудит изменений
CREATE TABLE rule_audit_log (
  id UUID PRIMARY KEY,
  rule_id UUID,
  rule_type VARCHAR(50),
  action VARCHAR(50),
  old_value JSONB,
  new_value JSONB,
  changed_by VARCHAR(255),
  timestamp TIMESTAMP DEFAULT NOW()
);

-- Индексы
CREATE INDEX idx_rule_audit_timestamp ON rule_audit_log(timestamp DESC);
CREATE INDEX idx_rule_audit_rule_id ON rule_audit_log(rule_id);
```

---

### 5. **Message Broker** 📬

#### Назначение
Асинхронная передача сообщений между сервисами. Гарантирует надежность и масштабируемость.

#### Выбор: Apache Kafka
```
Apache Kafka - приоритет: масштабируемость, история, высокая пропускная способность
```

#### Топология топиков (Kafka)
```
Topics:
├── input_events              (input для Filtering Service)
├── dedup_events              (output Filtering → input Deduplication)
├── enrichment_events         (output Deduplication → input Enrichment)
├── processed_events          (output Enrichment → final output)
└── config_updates            (для event-driven hot reload)

Consumer Groups:
├── filtering-service         (читает input_events)
├── dedup-service             (читает dedup_events)
├── enrichment-service        (читает enrichment_events)
└── config-watchers           (читают config_updates)
```

#### Message Format
```json
{
  "id": "msg-12345",
  "timestamp": "2025-12-14T14:55:00Z",
  "source": "api-gateway",
  "correlation_id": "corr-98765",
  "trace_id": "trace-54321",
  "payload": {...},
  "metadata": {
    "version": 1,
    "content_type": "application/json",
    "encoding": "utf-8"
  }
}
```

---

## Взаимодействие между сервисами

### Message Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ Message Lifecycle                                                │
└─────────────────────────────────────────────────────────────────┘

Stage 1: Filtering Service
├─ Read: input_events queue
├─ Process: Apply filtering rules
├─ Check: Rules cache (RWMutex protected)
├─ Decision: Pass or Filter
├─ Output: dedup_events queue
└─ Metrics: filtering.messages.total, filtering.passed, filtering.filtered

Stage 2: Deduplication Service
├─ Read: dedup_events queue
├─ Process: Hash message and check Redis
├─ Check: Redis with TTL window
├─ Decision: Unique or Duplicate
├─ Output: enrichment_events queue
└─ Metrics: dedup.unique, dedup.duplicate, dedup.cache_hits

Stage 3: Enrichment Service
├─ Read: enrichment_events queue
├─ Process: Apply enrichment rules
├─ Fetch: Data from external sources (with cache)
├─ Transform: Enrich message with data
├─ Output: processed_events queue
└─ Metrics: enrichment.processed, enrichment.errors, enrichment.cache_hit_ratio

Stage 4: Output
├─ Read: processed_events queue
├─ Use: By downstream consumers
└─ Audit: Optional audit logging

Error Handling:
├─ Stage 1-3 error → send to corresponding DLQ
├─ DLQ message → retry handler or manual review
└─ Max retries → system_dlq (final dead letter)
```

### Management Service Integration

```
┌──────────────────────────────┐
│   Management Service (REST)  │
└──────────────┬───────────────┘
               │
       ┌───────┴───────┬─────────────┬─────────────┐
       │               │             │             │
       ▼               ▼             ▼             ▼
┌────────────┐  ┌────────────┐ ┌────────────┐ ┌────────────┐
│ PostgreSQL │  │ PostgreSQL │ │ PostgreSQL │ │ RabbitMQ   │
│ (Filtering)│  │(Management)│ │(Versioning)│ │(Broadcast) │
└────────────┘  └────────────┘ └────────────┘ └────────────┘
       │               │             │             │
       └───────────────┴─────────────┴─────────────┘
                       │
                  ┌────▼────┐
                  │ Services│
                  │ Read    │
                  │ Config  │
                  └─────────┘
```

### Hot Reload Flow

```
┌─────────────────────────────┐
│  Admin updates rule via API │
│  PUT /api/v1/rules/filtering│
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│ Management Service          │
│ 1. Validate rule            │
│ 2. Save to PostgreSQL       │
│ 3. Log to audit table       │
│ 4. Send notification        │
└────────────┬────────────────┘
             │
      ┌──────┴───────────────┐
      │                      │
      ▼                      ▼
┌──────────────┐    ┌─────────────────────┐
│ Polling      │    │ Event-driven        │
│ (Option 1)   │    │ (Option 2)          │
│              │    │                     │
│ Service      │    │ Service subscribe   │
│ reads file   │    │ to RabbitMQ message │
│ every 5 sec  │    │ "config.updated"    │
│              │    │                     │
│ SIGHUP       │    │ Immediate update    │
│ (Option 3)   │    │                     │
│              │    │                     │
│ Service      │    │ <1 sec delay        │
│ reloads on   │    │                     │
│ signal       │    │                     │
│              │    │                     │
│ <1 min delay │    │                     │
└──────────────┘    └─────────────────────┘
             │
             ▼
┌─────────────────────────────┐
│ Filtering Service           │
│ 1. Acquire write lock       │
│ 2. Load new rules           │
│ 3. Compile regex patterns   │
│ 4. Release write lock       │
│ 5. Log info message         │
└─────────────────────────────┘
             │
             ▼
┌─────────────────────────────┐
│ New rules active            │
│ for all new messages        │
└─────────────────────────────┘
```

---

## Внутренние контракты

### 1. Filtering Service Contract

#### Input Message Type
```go
type Message map[string]interface{}

// Minimum required fields
type MinimalMessage struct {
    ID        string
    Timestamp time.Time
    Source    string
    // other fields...
}
```

#### Rule Evaluation Contract
```
Input:  Message, Rule
Output: bool (true = pass, false = filter)
Error:  error

Rule = {
    Field:    "field_name"
    Operator: "eq|contains|regex|gt|lt|in|range"
    Value:    interface{}
    Enabled:  bool
}

Contract:
- Must handle missing fields gracefully (return false)
- Must handle type mismatches (log warning, return false)
- Must be thread-safe (RWMutex on rules cache)
- Must not modify input message
- Must cache compiled regex patterns
```

#### Exported Functions
```go
func (fs *FilteringService) Process(ctx context.Context, msg Message) (bool, error)
func (fs *FilteringService) GetRules() []Rule
func (fs *FilteringService) UpsertRule(ctx context.Context, rule Rule) error
func (fs *FilteringService) DeleteRule(ctx context.Context, ruleID string) error
func (fs *FilteringService) ReloadRules() error
```

---

### 2. Deduplication Service Contract

#### Input Message Type (same as output of Filtering)
```go
type Message map[string]interface{} {
    "id": string (required)
    "timestamp": time.Time (required)
    "source": string (required)
    // fields from filtering...
    "filters_applied": {
        "rule_ids": []string
        "passed_at": time.Time
    }
}
```

#### Deduplication Contract
```
Input:  Message
Output: bool (true = unique, false = duplicate), error

Contract:
- Must extract configurable fields for hashing
- Must use consistent hash algorithm (MD5 or SHA256)
- Must set Redis TTL on insertion
- Must handle Redis connection failures gracefully
- Must be idempotent (repeated calls return same result)
- Must not modify input message
- Must update cache hit rate metric
```

#### Exported Functions
```go
func (ds *DeduplicationService) Process(ctx context.Context, msg Message) (bool, error)
func (ds *DeduplicationService) UpdateWindow(window time.Duration)
func (ds *DeduplicationService) GetCacheStats() CacheStats
func (ds *DeduplicationService) ClearCache(ctx context.Context) error
```

---

### 3. Enrichment Service Contract

#### Input Message Type
```go
type Message map[string]interface{} {
    // all previous fields...
    "filters_applied": {...}
    "deduplication": {...}
}
```

#### Enrichment Contract
```
Input:  Message, Rules
Output: EnrichedMessage, error

EnrichedMessage = {
    ...original message fields...
    "enrichment": {
        "data": {...}
        "rules_applied": []string
        "cache_hit": bool
        "enriched_at": time.Time
    }
}

Contract:
- Must not block pipeline on enrichment errors
- Must respect timeout for external API calls
- Must cache results with configurable TTL
- Must support fallback values
- Must handle partial enrichment (not all rules succeed)
- Must be thread-safe
- Must log all failures with details
- Must add enrichment metadata to message
```

#### Exported Functions
```go
func (es *EnrichmentService) Process(ctx context.Context, msg Message) (*Message, error)
func (es *EnrichmentService) GetRules() []EnrichmentRule
func (es *EnrichmentService) UpsertRule(ctx context.Context, rule EnrichmentRule) error
func (es *EnrichmentService) DeleteRule(ctx context.Context, ruleID string) error
func (es *EnrichmentService) ClearCache(ctx context.Context) error
```

---

### 4. Message Broker Contract

#### Producer Contract
```
func Publish(ctx context.Context, queue string, msg interface{}) error

Contract:
- Must serialize msg to JSON
- Must ensure queue exists
- Must handle connection failures with retry
- Must support acknowledgements
- Must timeout configurable (default 30s)
- Must add message metadata (timestamp, correlation_id)
```

#### Consumer Contract
```
func Consume(ctx context.Context, queue string) (<-chan Message, error)

Contract:
- Must support multiple concurrent consumers
- Must handle backpressure
- Must support manual/auto acknowledgement
- Must support error handling and DLQ routing
- Must preserve message ordering (per partition if Kafka)
- Must support graceful shutdown
```

---

### 5. Management Service Contract

#### Rule Validation Contract
```go
func ValidateFilteringRule(rule Rule) error

Validations:
- Field must not be empty
- Operator must be in predefined list
- Value must be compatible with operator
- Priority must be >= 0
- Name must be unique (or return version)

Returns:
- nil if valid
- ValidationError with detailed message if invalid
```

#### CRUD Contract
```
Create:
  Input:  Rule
  Output: Rule (with ID, timestamps), error
  
Read:
  Input:  ID
  Output: Rule, error
  
Update:
  Input:  ID, Rule
  Output: Rule (updated), error
  
Delete:
  Input:  ID
  Output: error

Contract:
- Must enforce permissions (RBAC optional)
- Must support soft deletes (or hard delete with audit trail)
- Must maintain version history
- Must notify services about changes
- Must validate before saving
- Must use transactions
```

---

### 6. Service-to-Service Communication

#### Health Check Contract
```
Endpoint: GET /health
Response: {
  "status": "healthy|degraded|unhealthy",
  "timestamp": time.Time,
  "uptime": duration,
  "database": "connected|disconnected",
  "broker": "connected|disconnected",
  "errors": []string
}
```

#### Metrics Export Contract
```
Endpoint: GET /metrics
Format:   Prometheus text format
Metrics:
  - {service}.messages.total (counter)
  - {service}.messages.success (counter)
  - {service}.messages.failed (counter)
  - {service}.messages.duration_ms (histogram)
  - {service}.cache.hit_rate (gauge)
  - {service}.queue.depth (gauge)
```

---

### 7. Error Handling Contract

#### Error Types
```go
// Recoverable errors (retry possible)
type RetryableError struct { ... }

// Non-recoverable errors (send to DLQ)
type FatalError struct { ... }

// Validation errors
type ValidationError struct { ... }

// Timeout errors
type TimeoutError struct { ... }

// Configuration errors
type ConfigError struct { ... }
```

#### Error Response Format
```json
{
  "error": "error message",
  "error_code": "ERROR_CODE",
  "timestamp": "2025-12-14T14:55:00Z",
  "trace_id": "trace-54321",
  "details": {
    "field": "value",
    "reason": "why it failed"
  }
}
```

#### Retry Policy
```
Max Retries:     3
Backoff:         exponential (1s, 2s, 4s)
Timeout:         30 seconds per retry
Circuit Breaker: open after 5 consecutive failures (reset after 1 min)
```

---

