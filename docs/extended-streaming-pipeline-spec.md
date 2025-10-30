# Расширенная спецификация StreamingPipeline

## Базовая спецификация

```yaml
apiVersion: etl.platform.io/v1
kind: StreamingPipeline
metadata:
  name: advanced-streaming-pipeline
  namespace: data-processing

spec:
  # Глобальные настройки
  config:
    processing_mode: "streaming"
    checkpoint_interval: "30s"
    watermark_interval: "10s"
    late_data_handling: "drop"  # drop, process, dead_letter
    parallelism: 4
    max_parallelism: 16
    
  # Источник данных
  source:
    type: kafka
    config:
      brokers: ["kafka-1:9092", "kafka-2:9092", "kafka-3:9092"]
      topic: "user.events.raw"
      consumer_group: "etl-streaming-v1"
      start_offset: "earliest"  # earliest, latest, timestamp
      security:
        protocol: "SASL_SSL"
        credentials_secret: "kafka_credentials"
      
  # Конвейер обработки данных
  processing:
    # Настройки окна (если необходимо)
    window:
      type: "tumbling"
      duration: "5m"
      allowed_lateness: "1m"
    
    # Этапы обработки данных
    stages:
      # Этап 1: Десериализация
      - name: "deserialize"
        type: "deserialize"
        config:
          format: "json"
          schema_registry:
            enabled: true
            url: "http://schema-registry:8081"
        resources:
          cpu: "200m"
          memory: "256Mi"
        scaling:
          min_replicas: 2
          max_replicas: 8
          target_cpu: 70
      
      # Этап 2: Фильтрация с поддержкой динамических правил
      - name: "filter"
        type: "filter"
        config:
          # Статические правила фильтрации
          conditions:
            - field: "event_type"
              operator: "in"
              values: ["click", "purchase", "signup"]
            - field: "user_id"
              operator: "not_null"
          # Настройки для динамических правил
          dynamic_rules:
            enabled: true
            api_endpoint: "/api/v1/filter-rules"
            refresh_interval: "30s"
            rule_store: "redis"
        # Указываем что этот этап критичен - нужен checkpoint
        checkpoint: true
        # Специализированная БД для хранения правил фильтрации
        database:
          type: "redis"
          config:
            host: "redis-filter"
            port: 6379
            database: 0
            credentials_secret: "redis_filter_credentials"
        resources:
          cpu: "300m"
          memory: "512Mi"
        scaling:
          min_replicas: 2
          max_replicas: 12
          target_cpu: 75
      
      # Этап 3: Дедупликация
      - name: "deduplicate"
        type: "deduplicate"
        config:
          # Настройки дедупликации
          key_fields: ["event_id", "user_id", "timestamp"]
          time_window: "1h"
          algorithm: "bloom_filter"  # exact, bloom_filter, probabilistic
          # Динамические правила дедупликации
          dynamic_rules:
            enabled: true
            api_endpoint: "/api/v1/deduplication-rules"
            refresh_interval: "30s"
        # Критичный этап для точности данных
        sensitive: true
        # Специализированная БД для дедупликации
        database:
          type: "cassandra"
          config:
            hosts: ["cassandra-dedupe-1", "cassandra-dedupe-2"]
            keyspace: "deduplication"
            replication_factor: 3
            credentials_secret: "cassandra_dedupe_credentials"
        resources:
          cpu: "500m"
          memory: "1Gi"
        scaling:
          min_replicas: 3
          max_replicas: 15
          target_cpu: 80
      
      # Этап 4: Обогащение данных
      - name: "enrich"
        type: "enrich"
        config:
          # Источники для обогащения
          enrichment_sources:
            - name: "user_profiles"
              type: "database"
              connection: "postgresql://user-db:5432/profiles"
              cache_ttl: "300s"
            - name: "geo_location"
              type: "api"
              endpoint: "http://geo-service:8080/location"
              timeout: "2s"
              retry_count: 2
          # Правила обогащения
          rules:
            - source: "user_profiles"
              join_key: "user_id"
              select_fields: ["age", "region", "subscription_tier"]
            - source: "geo_location"
              condition: "ip_address IS NOT NULL"
              join_key: "ip_address"
              select_fields: ["country", "city"]
          # Динамические правила обогащения
          dynamic_rules:
            enabled: true
            api_endpoint: "/api/v1/enrichment-rules"
            refresh_interval: "60s"
        # Специализированная БД для кеширования данных обогащения
        database:
          type: "mongodb"
          config:
            uri: "mongodb://mongo-enrich-cluster:27017/enrichment"
            collection: "cache"
            credentials_secret: "mongo_enrich_credentials"
        resources:
          cpu: "600m"
          memory: "1.5Gi"
        scaling:
          min_replicas: 2
          max_replicas: 10
          target_cpu: 70
      
      # Этап 5: Агрегация (финальный)
      - name: "aggregate"
        type: "aggregate"
        config:
          group_by: ["user_id", "event_type", "region"]
          aggregations:
            - field: "amount"
              function: "sum"
              alias: "total_amount"
            - field: "event_count"
              function: "count"
              alias: "event_count"
            - field: "timestamp"
              function: "max"
              alias: "last_event_time"
          output_mode: "update"
        resources:
          cpu: "400m"
          memory: "1Gi"
        scaling:
          min_replicas: 2
          max_replicas: 8
          target_cpu: 75

  # Промежуточные хранилища между критичными этапами
  intermediate_storage:
    - after_stage: "filter"  # После фильтрации
      type: "kafka"
      config:
        topic: "events.filtered"
        partitions: 12
        replication_factor: 3
        retention_ms: 86400000  # 24 часа
    
    - after_stage: "deduplicate"  # После дедупликации
      type: "kafka" 
      config:
        topic: "events.deduplicated"
        partitions: 12
        replication_factor: 3
        retention_ms: 86400000

  # Выходной поток
  sink:
    type: kafka
    config:
      brokers: ["kafka-1:9092", "kafka-2:9092", "kafka-3:9092"]
      topic: "events.processed"
      partitions: 16
      replication_factor: 3
      serialization: "json"
      acknowledgment: "all"
      compression_type: "lz4"

  # Мониторинг и алертинг
  monitoring:
    enabled: true
    metrics:
      - name: "throughput"
        type: "counter"
        labels: ["stage", "status"]
      - name: "processing_latency"
        type: "histogram"
        labels: ["stage"]
      - name: "error_rate"
        type: "gauge"
        labels: ["stage", "error_type"]
    
    alerts:
      - name: "high_error_rate"
        condition: "error_rate > 0.05"
        severity: "critical"
        notification: "slack://data-team"
      - name: "processing_lag"
        condition: "processing_latency_p99 > 30s"
        severity: "warning"
        notification: "email://ops@company.com"

  # Управление состоянием и восстановление
  state_management:
    backend: "rocksdb"  # rocksdb, filesystem, s3
    checkpoint_storage: "s3://checkpoints-bucket/streaming-pipeline"
    savepoint_directory: "s3://savepoints-bucket/streaming-pipeline"
    cleanup_policy: "delete"  # retain, delete
    incremental_checkpoints: true
    
  # Настройки безопасности
  security:
    encryption:
      at_rest: true
      in_transit: true
    authentication:
      type: "oauth2"
      provider: "keycloak"
    authorization:
      rbac_enabled: true
      service_account: "streaming-pipeline-sa"
```

## Конфигурация REST API для управления правилами

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: pipeline-rules-api
  namespace: data-processing
spec:
  selector:
    app: pipeline-rules-api
  ports:
    - name: http
      port: 8080
      targetPort: 8080
  type: ClusterIP

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pipeline-rules-api
  namespace: data-processing
spec:
  replicas: 3
  selector:
    matchLabels:
      app: pipeline-rules-api
  template:
    metadata:
      labels:
        app: pipeline-rules-api
    spec:
      containers:
      - name: rules-api
        image: streaming-platform/rules-api:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: rules-db-credentials
              key: url
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## Спецификация Kubernetes Operator

```yaml
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: streamingpipelines.etl.platform.io
spec:
  group: etl.platform.io
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              config:
                type: object
                properties:
                  processing_mode:
                    type: string
                    enum: ["streaming", "batch"]
                  checkpoint_interval:
                    type: string
                  watermark_interval:
                    type: string
                  late_data_handling:
                    type: string
                    enum: ["drop", "process", "dead_letter"]
              source:
                type: object
                properties:
                  type:
                    type: string
                  config:
                    type: object
              processing:
                type: object
                properties:
                  stages:
                    type: array
                    items:
                      type: object
                      properties:
                        name:
                          type: string
                        type:
                          type: string
                        checkpoint:
                          type: boolean
                        sensitive:
                          type: boolean
                        database:
                          type: object
                        resources:
                          type: object
                        scaling:
                          type: object
              intermediate_storage:
                type: array
                items:
                  type: object
                  properties:
                    after_stage:
                      type: string
                    type:
                      type: string
                    config:
                      type: object
              sink:
                type: object
          status:
            type: object
            properties:
              phase:
                type: string
                enum: ["Pending", "Running", "Failed", "Succeeded"]
              conditions:
                type: array
                items:
                  type: object
              stages:
                type: array
                items:
                  type: object
                  properties:
                    name:
                      type: string
                    status:
                      type: string
                    replicas:
                      type: integer
                    ready_replicas:
                      type: integer
  scope: Namespaced
  names:
    plural: streamingpipelines
    singular: streamingpipeline
    kind: StreamingPipeline
    shortNames:
    - sp
    - pipeline

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: streaming-pipeline-operator
rules:
- apiGroups: ["etl.platform.io"]
  resources: ["streamingpipelines"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["apps"]
  resources: ["deployments", "statefulsets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["services", "configmaps", "secrets", "persistentvolumeclaims"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["kafka.strimzi.io"]
  resources: ["kafkatopics"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: streaming-pipeline-operator
  namespace: etl-platform-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: streaming-pipeline-operator
  template:
    metadata:
      labels:
        app: streaming-pipeline-operator
    spec:
      serviceAccountName: streaming-pipeline-operator
      containers:
      - name: operator
        image: etl-platform/streaming-operator:v1.0.0
        command:
        - /manager
        args:
        - --leader-elect
        env:
        - name: WATCH_NAMESPACE
          value: ""
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

## Примеры API для динамического управления правилами

### API фильтрации
```http
# Получить текущие правила фильтрации
GET /api/v1/filter-rules?pipeline=advanced-streaming-pipeline

# Обновить правила фильтрации
POST /api/v1/filter-rules
Content-Type: application/json
{
  "pipeline": "advanced-streaming-pipeline",
  "rules": [
    {
      "id": "rule-001",
      "field": "event_type",
      "operator": "in",
      "values": ["click", "purchase", "signup", "view"],
      "enabled": true
    },
    {
      "id": "rule-002", 
      "field": "user_tier",
      "operator": "equals",
      "value": "premium",
      "enabled": true
    }
  ]
}
```

### API дедупликации
```http
# Получить настройки дедупликации
GET /api/v1/deduplication-rules?pipeline=advanced-streaming-pipeline

# Обновить правила дедупликации
PUT /api/v1/deduplication-rules
Content-Type: application/json
{
  "pipeline": "advanced-streaming-pipeline",
  "config": {
    "window_size": "2h",
    "key_fields": ["event_id", "user_id"],
    "algorithm": "exact",
    "cleanup_interval": "6h"
  }
}
```

### API обогащения данных
```http
# Получить правила обогащения
GET /api/v1/enrichment-rules?pipeline=advanced-streaming-pipeline

# Добавить новый источник обогащения
POST /api/v1/enrichment-rules/sources
Content-Type: application/json
{
  "pipeline": "advanced-streaming-pipeline",
  "source": {
    "name": "product_catalog",
    "type": "database",
    "connection": "postgresql://product-db:5432/catalog",
    "cache_ttl": "600s",
    "join_rules": [
      {
        "join_key": "product_id",
        "select_fields": ["category", "price", "brand"]
      }
    ]
  }
}
```