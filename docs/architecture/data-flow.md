# Data Flow Architecture

## Overview

```mermaid
graph TB
    subgraph "Data Sources"
        DB1[(Source Database)]
        FILES[File Systems]
        STREAMS[Message Streams]
        APIs[HTTP APIs]
        CLOUD[Cloud Storage]
    end

    subgraph "ETL Platform"
        INGESTION[Ingestion Layer]
        PROCESSING[Processing Layer]
        STORAGE[Storage Layer]
        MONITORING[Monitoring Layer]
    end

    subgraph "Data Destinations"
        DWH[(Data Warehouse)]
        LAKE[Data Lake]
        CACHE[(Cache)]
        ALERTS[Alert Systems]
        DASHBOARDS[Dashboards]
    end

    DB1 --> INGESTION
    FILES --> INGESTION
    STREAMS --> INGESTION
    APIs --> INGESTION
    CLOUD --> INGESTION

    INGESTION --> PROCESSING
    PROCESSING <--> STORAGE
    PROCESSING --> MONITORING

    STORAGE --> DWH
    STORAGE --> LAKE
    STORAGE --> CACHE
    MONITORING --> ALERTS
    MONITORING --> DASHBOARDS

    style INGESTION fill:#e3f2fd
    style PROCESSING fill:#e8f5e8
    style STORAGE fill:#fff3e0
    style MONITORING fill:#fce4ec
```

## Pipeline Data Flow

### End-to-End Pipeline Execution

```mermaid
flowchart LR
    subgraph "Source Systems"
        PG[(PostgreSQL)]
        MONGO[(MongoDB)]
        KAFKA[Kafka Stream]
        S3_SRC[S3 Files]
        API_SRC[REST APIs]
    end
    
    subgraph "Ingestion Stage"
        EXTRACT[Extract Operator]
        VALIDATE_SRC[Source Validation]
        BUFFER[Data Buffer]
    end

    subgraph "Processing Stage"
        FILTER[Filter Operator]
        TRANSFORM[Transform Operator]
        ENRICH[Enrichment Operator]
        AGGREGATE[Aggregate Operator]
        VALIDATE_PROC[Data Validation]
    end

    subgraph "Output Stage"
        FORMAT[Format Operator]
        PARTITION[Partition Operator]
        SINK[Sink Operator]
    end
    
    subgraph "Destination Systems"
        DWH[(Data Warehouse)]
        S3_DEST[S3 Data Lake]
        KAFKA_DEST[Kafka Topics]
        ELASTIC[Elasticsearch]
        REDIS_DEST[(Redis)]
    end
    
    PG --> EXTRACT
    MONGO --> EXTRACT
    KAFKA --> EXTRACT
    S3_SRC --> EXTRACT
    API_SRC --> EXTRACT
    
    EXTRACT --> VALIDATE_SRC
    VALIDATE_SRC --> BUFFER
    
    BUFFER --> FILTER
    FILTER --> TRANSFORM
    TRANSFORM --> ENRICH
    ENRICH --> AGGREGATE
    AGGREGATE --> VALIDATE_PROC
    
    VALIDATE_PROC --> FORMAT
    FORMAT --> PARTITION
    PARTITION --> SINK
    
    SINK --> DWH
    SINK --> S3_DEST
    SINK --> KAFKA_DEST
    SINK --> ELASTIC
    SINK --> REDIS_DEST
    
    style EXTRACT fill:#e3f2fd
    style TRANSFORM fill:#e8f5e8
    style VALIDATE_PROC fill:#fff3e0
    style SINK fill:#fce4ec
```

### Data Processing Patterns

#### Batch Processing Flow

```mermaid
sequenceDiagram
    participant Source as Data Source
    participant IE as Ingestion Engine
    participant PE as Processing Engine
    participant CS as Config Service
    participant Storage as Data Storage
    
    Note over Source,Storage: Batch Processing Flow
    
    Source->>IE: Read data in batches
    IE->>PE: Submit batch for processing
    PE->>CS: Get processing configuration
    CS-->>PE: Return config and schema
    
    loop For each batch
        PE->>PE: Apply transformations
        PE->>PE: Validate data quality
        PE->>PE: Apply business rules
    end
    
    PE->>Storage: Write processed batch
    PE->>IE: Confirm batch completion
    IE->>Source: Move to next batch
```

#### Stream Processing Flow

```mermaid
sequenceDiagram
    participant Stream as Message Stream
    participant IE as Ingestion Engine
    participant PE as Processing Engine
    participant Window as Window Manager
    participant Sink as Output Sink
    
    Note over Stream,Sink: Stream Processing Flow
    
    Stream->>IE: Continuous message flow
    IE->>PE: Forward messages
    
    loop Continuous Processing
        PE->>Window: Add to processing window
        Window->>Window: Accumulate messages
        
        alt Window Complete
            Window->>PE: Trigger window processing
            PE->>PE: Apply aggregations
            PE->>Sink: Output window results
        end
    end
```

## Control Flow

### Pipeline Lifecycle Management

```mermaid
stateDiagram-v2
    [*] --> Created
    Created --> Validating : Submit Pipeline
    Validating --> Validated : DSL Valid
    Validating --> Invalid : DSL Invalid
    Invalid --> [*]
    
    Validated --> Scheduled : Schedule Created
    Scheduled --> Triggered : Trigger Fired
    Triggered --> Running : Resources Allocated
    
    state Running {
        [*] --> Initializing
        Initializing --> Executing
        Executing --> Completing
        Completing --> [*]
        
        Executing --> Failed : Stage Failure
        Failed --> Retrying : Auto Retry
        Retrying --> Executing : Retry Attempt
        Retrying --> Failed : Max Retries Exceeded
    }
    
    Running --> Completed : Success
    Running --> Failed : Failure
    Running --> Cancelled : User Cancellation
    
    Completed --> [*]
    Failed --> [*]
    Cancelled --> [*]
    
    Failed --> Scheduled : Schedule Active
    Completed --> Scheduled : Schedule Active
```
