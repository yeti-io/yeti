# Отсутствующие интеграционные тесты

## Критичные пробелы

### 1. Enrichment Service - НЕТ ТЕСТОВ ВООБЩЕ ❌
- Process с различными source types (api, database, cache, mongodb, postgresql)
- Error handling стратегии:
  - `fail` - должно возвращать ошибку
  - `skip_rule` - должно пропускать правило
  - `skip_field` - должно пропускать поле
- Fallback values
- Transformations с CEL выражениями
- Кэширование (TTL, проверка кэша)
- Circuit breaker (если настроен)
- ReloadRules

### 2. Management Service - НЕТ ТЕСТОВ ВООБЩЕ ❌
- CRUD для Filtering Rules (частично есть в repository тестах, но нет в service)
- CRUD для Enrichment Rules
- Валидация CEL выражений при создании/обновлении
- Версионирование правил
- Audit logs (создание, чтение)
- Config events (публикация)
- UpdateDeduplicationConfig
- GetDeduplicationConfig

## Дополнительные кейсы для существующих тестов

### Filtering Service
- ❌ Fallback стратегии при ошибках CEL:
  - `FallbackAllow` - должно пропускать сообщение при ошибке
  - `FallbackDeny` - должно отклонять сообщение при ошибке
- ❌ Невалидные CEL выражения (должна быть ошибка)
- ❌ Контекст с таймаутом
- ❌ Контекст с отменой
- ❌ Пустой список правил
- ❌ Правила с разными приоритетами (проверка порядка применения)

### Deduplication Service
- ❌ Fallback стратегии при ошибках Redis:
  - `FallbackAllow` - должно возвращать true при ошибке Redis
  - `FallbackDeny` - должно возвращать false при ошибке Redis
- ❌ Разные hash алгоритмы (md5, sha256)
- ❌ Ошибки при вычислении hash (несуществующие поля)
- ❌ Контекст с таймаутом
- ❌ Контекст с отменой
- ❌ Пустые fields_to_hash (должны использоваться дефолтные)

### Management Repository
- ❌ Enrichment Rules CRUD (если есть методы)
- ❌ Версионирование (RuleVersion CRUD)
- ❌ Audit Logs (создание, чтение)

### Filtering Repository
- ❌ Пустой результат (уже есть)
- ❌ Правила с одинаковым приоритетом (проверка сортировки по created_at)

### Enrichment Repository
- ❌ Пустой результат (уже есть)
- ❌ Правила с одинаковым приоритетом (проверка сортировки)

### Deduplication Repository
- ✅ Все основные кейсы покрыты

## Edge cases и граничные условия

- ❌ Очень длинные сообщения
- ❌ Специальные символы в данных
- ❌ Unicode символы
- ❌ Null/empty значения в полях
- ❌ Вложенные структуры в payload
- ❌ Большое количество правил (performance)
- ❌ Одновременные обновления правил (concurrency)

## Рекомендации

1. **Приоритет 1 (критично):**
   - Enrichment Service тесты
   - Management Service тесты
   - Fallback стратегии для Filtering и Deduplication

2. **Приоритет 2 (важно):**
   - Error handling для Enrichment
   - Валидация CEL выражений
   - Контекст таймауты/отмена

3. **Приоритет 3 (желательно):**
   - Edge cases
   - Performance тесты
   - Concurrency тесты
