# 📁 Структура проекта

## Полная структура каталогов и файлов

```
data-pipeline/
├── cmd/                                      # Точки входа приложений
│   ├── filtering-service/
│   │   └── main.go                          # Входная точка Filtering Service
│   ├── dedup-service/
│   │   └── main.go                          # Входная точка Deduplication Service
│   ├── enrichment-service/
│   │   └── main.go                          # Входная точка Enrichment Service
│   └── management-service/
│       └── main.go                          # Входная точка Management Service
├── internal/                                 # Приватный код (не может быть импортирован извне)
│   ├── filtering/                           # Модуль фильтрации
│   │   ├── service.go                       # Основная логика фильтрации
│   │   ├── repository.go                    # PostgreSQL операции
│   │   ├── models.go                        # Структуры данных
│   │   ├── handler.go                       # HTTP обработчики
│   │   ├── validator.go                     # Валидация правил
│   │   ├── service_test.go                  # Unit тесты
│   │   └── repository_test.go
│   ├── deduplication/                       # Модуль дедубликации
│   │   ├── service.go                       # Основная логика дедубликации
│   │   ├── repository.go                    # Redis операции
│   │   ├── models.go                        # Структуры данных
│   │   ├── hasher.go                        # Вычисление хешей
│   │   ├── service_test.go
│   │   └── hasher_test.go
│   ├── enrichment/                          # Модуль обогащения
│   │   ├── service.go                       # Основная логика обогащения
│   │   ├── repository.go                    # MongoDB операции
│   │   ├── models.go                        # Структуры данных
│   │   ├── provider/                        # Провайдеры данных
│   │   │   ├── api_provider.go              # HTTP API провайдер
│   │   │   ├── database_provider.go         # Database провайдер
│   │   │   ├── cache_provider.go            # Cache провайдер
│   │   │   └── models.go
│   │   ├── service_test.go
│   │   └── provider_test.go
│   ├── management/                          # Management API модуль
│   │   ├── service.go                       # Бизнес логика управления
│   │   ├── handler.go                       # HTTP обработчики
│   │   ├── models.go                        # Структуры запросов/ответов
│   │   ├── validator.go                     # Валидация правил
│   │   ├── notifier.go                      # Уведомление сервисов об изменениях
│   │   ├── repository.go                    # PostgreSQL для хранения версий
│   │   ├── service_test.go
│   │   └── handler_test.go
│   ├── broker/                              # Работа с Message Broker
│   │   ├── consumer.go                      # Потребление сообщений
│   │   ├── producer.go                      # Отправка сообщений
│   │   ├── models.go                        # Message структуры
│   │   ├── factory.go                       # Factory для создания broker
│   │   └── *_test.go
│   ├── pipeline/                            # Pipeline оркестрация
│   │   ├── executor.go                      # Исполнитель pipeline
│   │   ├── stage.go                         # Этап обработки
│   │   ├── models.go                        # Структуры pipeline
│   │   └── executor_test.go
│   ├── config/                              # Управление конфигурацией
│   │   ├── config.go                        # Загрузка и парсинг конфига
│   │   ├── watcher.go                       # Hot reload механизм
│   │   ├── validator.go                     # Валидация конфига
│   │   ├── models.go                        # Структуры конфига
│   │   └── config_test.go
│   ├── logger/                              # Логирование
│   │   ├── logger.go                        # Инициализация Zap logger
│   │   ├── middleware.go                    # Middleware для логирования
│   │   └── fields.go                        # Предопределенные поля
│   ├── storage/                             # Абстракции для хранилищ
│   │   ├── postgres.go                      # PostgreSQL подключение
│   │   ├── redis.go                         # Redis подключение
│   │   ├── mongodb.go                       # MongoDB подключение
│   │   └── migrations.go                    # Управление миграциями
│   └── middleware/                          # HTTP middleware
│       ├── cors.go                          # CORS
│       ├── auth.go                          # Аутентификация/авторизация
│       ├── request_id.go                    # Request ID трейсинг
│       ├── logging.go                       # Request/response логирование
│       └── metrics.go                       # Сбор метрик
├── pkg/                                     # Публичные утилиты (могут использоваться извне)
│   ├── errors/                              # Кастомные ошибки
│   │   ├── errors.go                        # Определения ошибок
│   │   └── codes.go                         # Коды ошибок
│   ├── metrics/                             # Prometheus метрики
│   │   ├── metrics.go                       # Инициализация и экспорт
│   │   ├── counters.go                      # Счетчики
│   │   ├── gauges.go                        # Датчики
│   │   └── histograms.go                    # Гистограммы
│   ├── models/                              # Общие модели данных
│   │   ├── message.go                       # Message структура
│   │   ├── rule.go                          # Rule структуры
│   │   └── common.go                        # Общие типы
│   ├── utils/                               # Утилиты
│   │   ├── string.go                        # Работа со строками
│   │   ├── json.go                          # JSON утилиты
│   │   ├── validation.go                    # Валидация
│   │   ├── retry.go                         # Retry логика
│   │   ├── time.go                          # Временные утилиты
│   │   └── hash.go                          # Хеширование
│   └── health/                              # Health check
│       ├── health.go                        # Health status
│       └── checker.go                       # Health checkers
├── migrations/                              # Database миграции
│   ├── postgres/
│   │   ├── 001_init_schema.up.sql          # Initial schema (merged all migrations)
│   │   └── 001_init_schema.down.sql        # Rollback
│   └── mongodb/
│       └── 001_init_enrichment_rules.js
├── config/                                  # Конфигурационные файлы
│   ├── config.base.yaml                     # Базовая конфигурация
│   ├── config.dev.yaml                      # Разработка
│   ├── config.staging.yaml                  # Staging
│   ├── config.prod.yaml                     # Production
│   ├── rules.filtering.yaml                 # Правила фильтрации
│   ├── rules.enrichment.yaml                # Правила обогащения
│   └── logging.yaml                         # Конфиг логирования
├── docker/                                  # Docker файлы
│   ├── Dockerfile.filtering                 # Filtering Service
│   ├── Dockerfile.dedup                     # Dedup Service
│   ├── Dockerfile.enrichment                # Enrichment Service
│   ├── Dockerfile.management                # Management Service
│   └── .dockerignore
├── scripts/                                 # Утилит скрипты
│   ├── setup-db.sh                          # Инициализация БД
│   ├── generate-migrations.sh               # Генерация миграций
│   ├── health-check.sh                      # Health check скрипт
│   └── load-test.sh                         # Load testing
├── tests/                                   # Integration и e2e тесты
│   ├── integration/
│   │   ├── filtering_test.go
│   │   ├── dedup_test.go
│   │   ├── enrichment_test.go
│   │   └── pipeline_test.go
│   ├── e2e/
│   │   ├── full_pipeline_test.go
│   │   └── api_test.go
│   └── fixtures/
│       ├── messages.json
│       ├── rules.json
│       └── data.yaml
├── docs/                                    # Документация
│   ├── API.md                               # API документация
│   ├── DEPLOYMENT.md                        # Deployment guide
│   ├── MONITORING.md                        # Monitoring guide
│   └── TROUBLESHOOTING.md                   # Troubleshooting
├── prometheus/                              # Prometheus конфигурация
│   └── prometheus.yml
├── grafana/                                 # Grafana конфигурация
│   └── dashboards/
│       ├── pipeline-overview.json
│       ├── filtering-metrics.json
│       ├── dedup-metrics.json
│       └── enrichment-metrics.json
├── docker-compose.yml                       # Docker Compose для локальной разработки
├── docker-compose.prod.yml                  # Docker Compose для production
├── Makefile                                 # Make задачи
├── go.mod                                   # Go модули
├── go.sum
├── .env.example                             # Пример переменных окружения
├── .gitignore
├── README.md                                # Основная документация
└── LICENSE
```

---

## Описание ключевых директорий

### `/cmd`
Каждый микросервис имеет отдельную папку с `main.go`. Это точка входа приложения.

```
cmd/filtering-service/main.go:
  - инициализирует логирование
  - загружает конфиг
  - подключается к БД
  - создает service
  - запускает HTTP сервер
  - создает обработчик сообщений из broker
  - обрабатывает graceful shutdown
```

### `/internal`
Приватный код, не доступный для импорта из других Go модулей. Организован по доменам (filtering, deduplication и т.д.).

```
internal/filtering/:
  - service.go      - бизнес логика (Process, GetRules, UpsertRule)
  - repository.go   - работа с PostgreSQL
  - models.go       - структуры (Rule, FilteringConfig)
  - handler.go      - HTTP обработчики (для Management API)
  - validator.go    - валидация правил
  - *_test.go       - unit тесты
```

### `/pkg`
Публичный, переиспользуемый код. Может быть импортирован другими проектами.

```
pkg/errors/:
  - errors.go - кастомные ошибки (ValidationError, RetryableError и т.д.)

pkg/metrics/:
  - metrics.go - Prometheus метрики (counters, gauges, histograms)

pkg/utils/:
  - retry.go - утилиты для retry логики
  - validation.go - функции валидации
```

### `/migrations`
SQL миграции для инициализации и обновления БД схемы.

```
postgres/:
  001_init_schema.up.sql      - создание всех таблиц (filtering_rules, rule_versions, rule_audit_logs, api_access_logs)
  001_init_schema.down.sql    - откат всех таблиц

mongodb/:
  001_init_enrichment_rules.js - инициализация коллекций и индексов для правил обогащения
```

### `/config`
YAML конфигурационные файлы для разных окружений.

```
config.base.yaml       - базовые параметры
config.dev.yaml        - переопределение для development
config.prod.yaml       - переопределение для production
rules.filtering.yaml   - правила фильтрации (могут меняться в runtime)
```

### `/docker`
Dockerfile для каждого микросервиса (мультистадийная сборка).

### `/tests`
Integration и e2e тесты.

```
integration/:
  - тесты с реальными БД (Docker containers)
  - проверка взаимодействия компонентов

e2e/:
  - полный pipeline от input до output
  - проверка end-to-end функциональности
```

---

## Соглашения по организации

### 1. Структура пакета (domain-driven)
```
internal/filtering/
├── service.go           # Main business logic
├── repository.go        # Data access layer
├── models.go            # Domain models
├── handler.go           # HTTP handlers (if applicable)
├── validator.go         # Domain validation
└── *_test.go            # Tests
```

### 2. Именование файлов
- `*_test.go` - unit тесты
- `*_integration_test.go` - integration тесты
- `service.go` - основная бизнес логика
- `repository.go` - data access
- `handler.go` - HTTP обработчики
- `models.go` - структуры данных
- `validator.go` - валидация

### 3. Импорты в коде
```go
// internal/filtering/service.go
package filtering

import (
    // стандартная библиотека
    "context"
    "errors"

    // внешние зависимости
    "github.com/lib/pq"
    
    // проект
    "data-pipeline/internal/config"
    "data-pipeline/pkg/errors" // публичные ошибки
    "data-pipeline/pkg/metrics"
)
```

### 4. Package initialization
```go
// internal/filtering/service.go
func New(repo Repository, metrics *metrics.Metrics, logger logger.Logger) *Service {
    return &Service{
        repo:    repo,
        metrics: metrics,
        logger:  logger,
    }
}

// Test
func TestFiltering(t *testing.T) {
    mockRepo := &MockRepository{}
    service := New(mockRepo, mockMetrics, mockLogger)
    // assertions...
}
```

### 5. Тестовые файлы
```go
// Один тестовый файл на один модуль
// internal/filtering/service_test.go
func TestProcess(t *testing.T) { ... }
func TestGetRules(t *testing.T) { ... }
func TestUpsertRule(t *testing.T) { ... }
```

---

## Best Practices

1. **Отделение concerns** - каждый файл отвечает за одно
2. **Interface-driven** - использовать interfaces для dependency injection
3. **Error handling** - explicit error handling, no panic in production code
4. **Testing** - 80%+ code coverage
5. **Logging** - структурированное логирование (Zap)
6. **Metrics** - метрики на всех критических местах
7. **Graceful shutdown** - корректное завершение goroutines и connections

---

