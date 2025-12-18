# Integration Tests

Интеграционные тесты для проверки работы компонентов системы Yeti с реальными базами данных.

## Структура

```
tests/integration/
├── setup.go                          # Настройка testcontainers
├── management_repository_test.go      # Тесты Management Repository
├── filtering_repository_test.go      # Тесты Filtering Repository
├── enrichment_repository_test.go     # Тесты Enrichment Repository
├── deduplication_repository_test.go  # Тесты Deduplication Repository
├── filtering_service_test.go         # Тесты Filtering Service
├── deduplication_service_test.go     # Тесты Deduplication Service
└── README.md                         # Документация

tests/fixtures/
├── filtering_rules.json              # Фикстуры для правил фильтрации
├── enrichment_rules.json             # Фикстуры для правил обогащения
└── messages.json                     # Фикстуры для сообщений
```

## Требования

1. **Docker** - должен быть установлен и запущен
2. **Go 1.25+** - для компиляции тестов
3. **testcontainers-go** - автоматически поднимает контейнеры

## Запуск тестов

### Все интеграционные тесты

```bash
go test ./tests/integration/... -v
```

### Конкретный тест

```bash
go test ./tests/integration/... -v -run TestManagementRepository_CreateFilteringRule
```

### Пропустить интеграционные тесты (short mode)

```bash
go test ./tests/integration/... -short
```

### С покрытием

```bash
go test ./tests/integration/... -cover
```

## Что тестируется

### Repositories

#### Management Repository
- ✅ Создание правил фильтрации
- ✅ Получение правила по ID
- ✅ Получение правила (not found)
- ✅ Список всех правил
- ✅ Обновление правила
- ✅ Удаление правила

#### Filtering Repository
- ✅ Получение активных правил
- ✅ Фильтрация только enabled правил
- ✅ Сортировка по priority и created_at

#### Enrichment Repository
- ✅ Получение активных правил
- ✅ Фильтрация только enabled правил
- ✅ Сортировка по priority

#### Deduplication Repository
- ✅ SetNX для уникальных ключей
- ✅ SetNX для дубликатов
- ✅ TTL для ключей
- ✅ Разные ключи
- ✅ Отмена контекста

### Services

#### Filtering Service
- ✅ Фильтрация сообщений (pass)
- ✅ Фильтрация сообщений (reject)
- ✅ Множественные правила
- ✅ Перезагрузка правил

#### Deduplication Service
- ✅ Обработка уникальных сообщений
- ✅ Обнаружение дубликатов
- ✅ Разные сообщения
- ✅ Кастомные поля для хеширования
- ✅ Обновление полей для хеширования

## Инфраструктура

Тесты используют [testcontainers-go](https://github.com/testcontainers/testcontainers-go) для автоматического поднятия:

- **PostgreSQL 15** - для хранения правил фильтрации
- **MongoDB 6** - для хранения правил обогащения
- **Redis 7** - для кэширования и дедупликации

Все контейнеры автоматически:
- Запускаются перед тестами
- Очищаются после тестов
- Изолированы друг от друга

## Фикстуры

Фикстуры находятся в `tests/fixtures/` и содержат примеры данных для тестирования:

- `filtering_rules.json` - примеры правил фильтрации
- `enrichment_rules.json` - примеры правил обогащения
- `messages.json` - примеры сообщений для обработки

## Примечания

- Тесты изолированы и могут запускаться параллельно
- Каждый тест получает свою инфраструктуру через `SetupTestInfra`
- Миграции PostgreSQL запускаются автоматически перед тестами
- Все тесты используют `context.Background()` для простоты
