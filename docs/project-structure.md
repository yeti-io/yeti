# YeTi Project Structure - Complete Guide

## Обзор структуры монорепо

YeTi организован как монорепо с использованием Bazel для управления зависимостями и сборкой. Структура чётко разделяет компоненты по функциональности и слоям архитектуры.

## Полная структура проекта

```
yeti-platform/
├── WORKSPACE                           # Bazel workspace configuration
├── MODULE.bazel                        # Bazel module dependencies (bzlmod)
├── .bazelrc                           # Bazel build configuration
├── .bazelversion                      # Lock Bazel version
├── BUILD.bazel                        # Root build file
├── README.md
├── LICENSE
├── .gitignore
│
├── yeti/                              # Main source code
│   ├── BUILD.bazel
│   │
│   ├── dataplane/                     # Data Plane компоненты
│   │   ├── BUILD.bazel
│   │   │
│   │   ├── executor/                  # Pipeline Executor (C++)
│   │   │   ├── BUILD.bazel
│   │   │   ├── main.cc
│   │   │   │
│   │   │   ├── engine/                # Pipeline execution engine
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── pipeline_engine.h
│   │   │   │   ├── pipeline_engine.cc
│   │   │   │   ├── event.h
│   │   │   │   ├── event.cc
│   │   │   │   └── config.h
│   │   │   │
│   │   │   ├── runtime/               # Stage runtime environment
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── stage_runtime.h
│   │   │   │   ├── stage_runtime.cc
│   │   │   │   ├── plugin_loader.h
│   │   │   │   └── plugin_loader.cc
│   │   │   │
│   │   │   ├── coordinator/           # Multi-stage coordination
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── coordinator.h
│   │   │   │   ├── coordinator.cc
│   │   │   │   └── backpressure.h
│   │   │   │
│   │   │   └── grpc/                  # gRPC server for control
│   │   │       ├── BUILD.bazel
│   │   │       ├── server.h
│   │   │       ├── server.cc
│   │   │       └── handlers.cc
│   │   │
│   │   ├── stages/                    # Stage implementations
│   │   │   ├── BUILD.bazel
│   │   │   │
│   │   │   ├── base/                  # Base classes and interfaces
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── stage.h            # Stage interface
│   │   │   │   ├── context.h          # Execution context
│   │   │   │   ├── lifecycle.h        # Init/Process/Shutdown
│   │   │   │   └── status.h           # Status codes
│   │   │   │
│   │   │   ├── io/                    # Input/Output stages
│   │   │   │   ├── BUILD.bazel
│   │   │   │   │
│   │   │   │   ├── sources/           # Source stages
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── kafka/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── kafka_source.h
│   │   │   │   │   │   ├── kafka_source.cc
│   │   │   │   │   │   └── config.h
│   │   │   │   │   ├── kinesis/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   └── kinesis_source.cc
│   │   │   │   │   ├── rabbitmq/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   └── rabbitmq_source.cc
│   │   │   │   │   ├── http/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   └── http_source.cc
│   │   │   │   │   └── file/
│   │   │   │   │       ├── BUILD.bazel
│   │   │   │   │       └── file_source.cc
│   │   │   │   │
│   │   │   │   └── sinks/             # Sink stages
│   │   │   │       ├── BUILD.bazel
│   │   │   │       ├── kafka/
│   │   │   │       │   ├── BUILD.bazel
│   │   │   │       │   └── kafka_sink.cc
│   │   │   │       ├── s3/
│   │   │   │       │   ├── BUILD.bazel
│   │   │   │       │   └── s3_sink.cc
│   │   │   │       ├── clickhouse/
│   │   │   │       │   ├── BUILD.bazel
│   │   │   │       │   └── clickhouse_sink.cc
│   │   │   │       ├── elasticsearch/
│   │   │   │       │   ├── BUILD.bazel
│   │   │   │       │   └── elasticsearch_sink.cc
│   │   │   │       └── http/
│   │   │   │           ├── BUILD.bazel
│   │   │   │           └── http_sink.cc
│   │   │   │
│   │   │   ├── processors/            # Processor stages
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── processor_stage.h
│   │   │   │   ├── processor_stage.cc
│   │   │   │   │
│   │   │   │   ├── core/              # Built-in processor plugins
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │
│   │   │   │   │   ├── filter/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── filter_plugin.h
│   │   │   │   │   │   ├── filter_plugin.cc
│   │   │   │   │   │   └── cel_evaluator.cc
│   │   │   │   │   │
│   │   │   │   │   ├── map/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── map_plugin.h
│   │   │   │   │   │   └── map_plugin.cc
│   │   │   │   │   │
│   │   │   │   │   ├── dedup/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── dedup_plugin.h
│   │   │   │   │   │   ├── dedup_plugin.cc
│   │   │   │   │   │   └── hash.cc
│   │   │   │   │   │
│   │   │   │   │   ├── enrich/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── enrich_plugin.h
│   │   │   │   │   │   ├── enrich_plugin.cc
│   │   │   │   │   │   ├── sources/
│   │   │   │   │   │   │   ├── database.cc
│   │   │   │   │   │   │   ├── redis.cc
│   │   │   │   │   │   │   └── http.cc
│   │   │   │   │   │   └── cache.cc
│   │   │   │   │   │
│   │   │   │   │   ├── validate/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── validate_plugin.h
│   │   │   │   │   │   └── validate_plugin.cc
│   │   │   │   │   │
│   │   │   │   │   ├── aggregate/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── aggregate_plugin.h
│   │   │   │   │   │   └── aggregate_plugin.cc
│   │   │   │   │   │
│   │   │   │   │   └── router/
│   │   │   │   │       ├── BUILD.bazel
│   │   │   │   │       ├── router_plugin.h
│   │   │   │   │       └── router_plugin.cc
│   │   │   │   │
│   │   │   │   ├── loader/            # Dynamic plugin loader
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── plugin_loader.h
│   │   │   │   │   ├── plugin_loader.cc
│   │   │   │   │   └── dlopen_loader.cc
│   │   │   │   │
│   │   │   │   ├── isolation/         # Plugin isolation modes
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── inprocess_runner.h
│   │   │   │   │   ├── inprocess_runner.cc
│   │   │   │   │   ├── isolated_runner.h
│   │   │   │   │   └── isolated_runner.cc
│   │   │   │   │
│   │   │   │   └── registry/          # Local plugin registry
│   │   │   │       ├── BUILD.bazel
│   │   │   │       ├── plugin_registry.h
│   │   │   │       └── plugin_registry.cc
│   │   │   │
│   │   │   └── custom/                # User-defined stages
│   │   │       ├── BUILD.bazel
│   │   │       ├── template/          # Template for new stages
│   │   │       │   ├── BUILD.bazel
│   │   │       │   ├── custom_stage.h
│   │   │       │   └── custom_stage.cc
│   │   │       └── examples/
│   │   │           ├── BUILD.bazel
│   │   │           └── json_parser_stage.cc
│   │   │
│   │   ├── transport/                 # Inter-stage transport
│   │   │   ├── BUILD.bazel
│   │   │   ├── transport.h            # Transport interface
│   │   │   │
│   │   │   ├── memory/                # In-process memory transport
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── memory_transport.h
│   │   │   │   └── memory_transport.cc
│   │   │   │
│   │   │   ├── kafka/                 # Kafka-backed transport
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── kafka_transport.h
│   │   │   │   └── kafka_transport.cc
│   │   │   │
│   │   │   ├── grpc/                  # gRPC streaming transport
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── grpc_transport.h
│   │   │   │   └── grpc_transport.cc
│   │   │   │
│   │   │   └── manager/               # Transport manager
│   │   │       ├── BUILD.bazel
│   │   │       ├── transport_manager.h
│   │   │       └── transport_manager.cc
│   │   │
│   │   └── common/                    # Common data plane libraries
│   │       ├── BUILD.bazel
│   │       │
│   │       ├── event/                 # Event model
│   │       │   ├── BUILD.bazel
│   │       │   ├── event.h
│   │       │   ├── event.cc
│   │       │   └── event.proto
│   │       │
│   │       ├── metrics/               # Metrics collection
│   │       │   ├── BUILD.bazel
│   │       │   ├── metrics_collector.h
│   │       │   ├── metrics_collector.cc
│   │       │   └── prometheus_exporter.cc
│   │       │
│   │       ├── tracing/               # Distributed tracing
│   │       │   ├── BUILD.bazel
│   │       │   ├── tracing_manager.h
│   │       │   ├── tracing_manager.cc
│   │       │   └── otel_integration.cc
│   │       │
│   │       ├── logging/               # Structured logging
│   │       │   ├── BUILD.bazel
│   │       │   ├── logger.h
│   │       │   └── logger.cc
│   │       │
│   │       └── config/                # Configuration management
│   │           ├── BUILD.bazel
│   │           ├── config_manager.h
│   │           └── config_manager.cc
│   │
│   ├── controlplane/                  # Control Plane компоненты
│   │   ├── BUILD.bazel
│   │   │
│   │   ├── controller/                # Main controller (Go)
│   │   │   ├── BUILD.bazel
│   │   │   ├── main.go
│   │   │   │
│   │   │   ├── cmd/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   └── root.go
│   │   │   │
│   │   │   ├── internal/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   │
│   │   │   │   ├── api/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── rest/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── server.go
│   │   │   │   │   │   ├── routes.go
│   │   │   │   │   │   ├── handlers/
│   │   │   │   │   │   │   ├── pipeline.go
│   │   │   │   │   │   │   ├── stage.go
│   │   │   │   │   │   │   └── plugin.go
│   │   │   │   │   │   └── middleware/
│   │   │   │   │   │       ├── auth.go
│   │   │   │   │   │       └── logging.go
│   │   │   │   │   │
│   │   │   │   │   └── grpc/
│   │   │   │   │       ├── BUILD.bazel
│   │   │   │   │       ├── server.go
│   │   │   │   │       └── services/
│   │   │   │   │           ├── config.go
│   │   │   │   │           └── executor.go
│   │   │   │   │
│   │   │   │   ├── service/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── pipeline.go
│   │   │   │   │   ├── stage.go
│   │   │   │   │   ├── plugin.go
│   │   │   │   │   ├── config.go
│   │   │   │   │   └── executor.go
│   │   │   │   │
│   │   │   │   ├── repository/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── interface.go
│   │   │   │   │   ├── postgres/
│   │   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   │   ├── pipeline.go
│   │   │   │   │   │   ├── stage.go
│   │   │   │   │   │   └── migrations/
│   │   │   │   │   │       └── *.sql
│   │   │   │   │   └── cache/
│   │   │   │   │       ├── BUILD.bazel
│   │   │   │   │       └── redis.go
│   │   │   │   │
│   │   │   │   └── models/
│   │   │   │       ├── BUILD.bazel
│   │   │   │       ├── pipeline.go
│   │   │   │       ├── stage.go
│   │   │   │       └── plugin.go
│   │   │   │
│   │   │   └── pkg/
│   │   │       └── BUILD.bazel
│   │   │
│   │   ├── compiler/                  # Pipeline compiler (Go)
│   │   │   ├── BUILD.bazel
│   │   │   ├── main.go
│   │   │   │
│   │   │   ├── internal/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   │
│   │   │   │   ├── parser/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── parser.go
│   │   │   │   │   ├── yaml_parser.go
│   │   │   │   │   └── ast.go
│   │   │   │   │
│   │   │   │   ├── validator/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── validator.go
│   │   │   │   │   ├── semantic.go
│   │   │   │   │   └── schema.go
│   │   │   │   │
│   │   │   │   ├── optimizer/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── optimizer.go
│   │   │   │   │   ├── passes/
│   │   │   │   │   │   ├── stage_fusion.go
│   │   │   │   │   │   ├── dead_code_elimination.go
│   │   │   │   │   │   └── transport_optimization.go
│   │   │   │   │   └── pass.go
│   │   │   │   │
│   │   │   │   └── codegen/
│   │   │   │       ├── BUILD.bazel
│   │   │   │       ├── codegen.go
│   │   │   │       ├── runtime_config.go
│   │   │   │       └── templates/
│   │   │   │
│   │   │   └── pkg/
│   │   │       └── BUILD.bazel
│   │   │
│   │   ├── registry/                  # Stage & Plugin registry (Go)
│   │   │   ├── BUILD.bazel
│   │   │   ├── main.go
│   │   │   │
│   │   │   ├── internal/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   │
│   │   │   │   ├── catalog/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   ├── stage_catalog.go
│   │   │   │   │   └── plugin_catalog.go
│   │   │   │   │
│   │   │   │   ├── versions/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   └── version_manager.go
│   │   │   │   │
│   │   │   │   └── metadata/
│   │   │   │       ├── BUILD.bazel
│   │   │   │       └── metadata_store.go
│   │   │   │
│   │   │   └── pkg/
│   │   │       └── BUILD.bazel
│   │   │
│   │   ├── plugin-manager/            # Plugin manager (Go)
│   │   │   ├── BUILD.bazel
│   │   │   ├── main.go
│   │   │   │
│   │   │   ├── internal/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   │
│   │   │   │   ├── loader/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   └── plugin_loader.go
│   │   │   │   │
│   │   │   │   ├── validator/
│   │   │   │   │   ├── BUILD.bazel
│   │   │   │   │   └── plugin_validator.go
│   │   │   │   │
│   │   │   │   └── marketplace/
│   │   │   │       ├── BUILD.bazel
│   │   │   │       ├── marketplace.go
│   │   │   │       ├── installer.go
│   │   │   │       └── registry_client.go
│   │   │   │
│   │   │   └── pkg/
│   │   │       └── BUILD.bazel
│   │   │
│   │   └── storage/                   # Storage layer (Go)
│   │       ├── BUILD.bazel
│   │       ├── postgres/
│   │       │   ├── BUILD.bazel
│   │       │   └── schema.sql
│   │       └── migrations/
│   │           └── BUILD.bazel
│   │
│   ├── operator/                      # Kubernetes Operator (Go)
│   │   ├── BUILD.bazel
│   │   ├── main.go
│   │   ├── Dockerfile
│   │   │
│   │   ├── api/
│   │   │   └── v1/
│   │   │       ├── BUILD.bazel
│   │   │       ├── pipeline_types.go
│   │   │       ├── stage_types.go
│   │   │       ├── plugin_types.go
│   │   │       ├── groupversion_info.go
│   │   │       └── zz_generated.deepcopy.go
│   │   │
│   │   ├── controllers/
│   │   │   ├── BUILD.bazel
│   │   │   ├── pipeline_controller.go
│   │   │   ├── stage_controller.go
│   │   │   └── plugin_controller.go
│   │   │
│   │   └── config/
│   │       ├── crd/
│   │       │   ├── bases/
│   │       │   │   ├── yeti.io_pipelines.yaml
│   │       │   │   ├── yeti.io_stages.yaml
│   │       │   │   └── yeti.io_plugins.yaml
│   │       │   └── kustomization.yaml
│   │       ├── rbac/
│   │       │   ├── role.yaml
│   │       │   ├── role_binding.yaml
│   │       │   └── service_account.yaml
│   │       └── manager/
│   │           ├── kustomization.yaml
│   │           └── manager.yaml
│   │
│   ├── api/                           # API Gateway
│   │   ├── BUILD.bazel
│   │   ├── gateway/
│   │   │   └── BUILD.bazel
│   │   ├── pipeline/
│   │   │   └── BUILD.bazel
│   │   ├── stages/
│   │   │   └── BUILD.bazel
│   │   ├── plugins/
│   │   │   └── BUILD.bazel
│   │   └── observability/
│   │       └── BUILD.bazel
│   │
│   ├── ui/                            # Web Console
│   │   ├── BUILD.bazel
│   │   ├── console/
│   │   │   ├── BUILD.bazel
│   │   │   ├── package.json
│   │   │   ├── tsconfig.json
│   │   │   ├── vite.config.ts
│   │   │   │
│   │   │   ├── public/
│   │   │   │   └── index.html
│   │   │   │
│   │   │   └── src/
│   │   │       ├── App.tsx
│   │   │       ├── main.tsx
│   │   │       │
│   │   │       ├── pages/
│   │   │       │   ├── pipelines/
│   │   │       │   │   ├── PipelineList.tsx
│   │   │       │   │   ├── PipelineDetail.tsx
│   │   │       │   │   └── PipelineCreate.tsx
│   │   │       │   ├── stages/
│   │   │       │   │   └── StageCatalog.tsx
│   │   │       │   ├── plugins/
│   │   │       │   │   ├── PluginMarketplace.tsx
│   │   │       │   │   └── PluginDetail.tsx
│   │   │       │   └── monitoring/
│   │   │       │       ├── Dashboard.tsx
│   │   │       │       └── Metrics.tsx
│   │   │       │
│   │   │       ├── components/
│   │   │       │   ├── pipeline-editor/
│   │   │       │   │   ├── VisualEditor.tsx
│   │   │       │   │   ├── YamlEditor.tsx
│   │   │       │   │   └── StageNode.tsx
│   │   │       │   ├── stage-config/
│   │   │       │   │   └── ConfigForm.tsx
│   │   │       │   └── shared/
│   │   │       │       ├── Layout.tsx
│   │   │       │       └── Navbar.tsx
│   │   │       │
│   │   │       ├── services/
│   │   │       │   ├── api.ts
│   │   │       │   ├── pipeline.ts
│   │   │       │   └── plugin.ts
│   │   │       │
│   │   │       └── lib/
│   │   │           ├── types.ts
│   │   │           └── utils.ts
│   │   │
│   │   └── BUILD.bazel
│   │
│   ├── cli/                           # CLI Tool (Go)
│   │   ├── BUILD.bazel
│   │   ├── main.go
│   │   │
│   │   ├── cmd/
│   │   │   ├── BUILD.bazel
│   │   │   ├── root.go
│   │   │   │
│   │   │   ├── pipeline/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── create.go
│   │   │   │   ├── list.go
│   │   │   │   ├── get.go
│   │   │   │   ├── update.go
│   │   │   │   ├── delete.go
│   │   │   │   ├── deploy.go
│   │   │   │   └── validate.go
│   │   │   │
│   │   │   ├── plugin/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── list.go
│   │   │   │   ├── install.go
│   │   │   │   ├── uninstall.go
│   │   │   │   └── search.go
│   │   │   │
│   │   │   ├── debug/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── logs.go
│   │   │   │   ├── trace.go
│   │   │   │   └── exec.go
│   │   │   │
│   │   │   └── config/
│   │   │       ├── BUILD.bazel
│   │   │       └── set.go
│   │   │
│   │   └── pkg/
│   │       ├── BUILD.bazel
│   │       ├── client/
│   │       │   └── client.go
│   │       └── output/
│   │           └── formatter.go
│   │
│   ├── sdk/                           # SDKs
│   │   ├── BUILD.bazel
│   │   │
│   │   ├── cpp/                       # C++ SDK
│   │   │   ├── BUILD.bazel
│   │   │   │
│   │   │   ├── stage/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── stage_sdk.h
│   │   │   │   ├── stage_base.h
│   │   │   │   └── examples/
│   │   │   │       └── custom_stage.cc
│   │   │   │
│   │   │   └── processor/
│   │   │       ├── BUILD.bazel
│   │   │       ├── plugin_sdk.h
│   │   │       ├── plugin_base.h
│   │   │       └── examples/
│   │   │           └── custom_plugin.cc
│   │   │
│   │   ├── go/                        # Go SDK
│   │   │   ├── BUILD.bazel
│   │   │   │
│   │   │   ├── client/
│   │   │   │   ├── BUILD.bazel
│   │   │   │   ├── client.go
│   │   │   │   └── pipeline_client.go
│   │   │   │
│   │   │   └── plugin/
│   │   │       ├── BUILD.bazel
│   │   │       └── plugin.go
│   │   │
│   │   ├── python/                    # Python SDK
│   │   │   ├── BUILD.bazel
│   │   │   ├── setup.py
│   │   │   │
│   │   │   └── yeti/
│   │   │       ├── __init__.py
│   │   │       ├── client.py
│   │   │       └── pipeline.py
│   │   │
│   │   └── types/                     # Common types
│   │       ├── BUILD.bazel
│   │       │
│   │       ├── proto/
│   │       │   ├── BUILD.bazel
│   │       │   ├── common.proto
│   │       │   ├── pipeline.proto
│   │       │   ├── stage.proto
│   │       │   └── plugin.proto
│   │       │
│   │       └── schema/
│   │           ├── BUILD.bazel
│   │           ├── pipeline.schema.json
│   │           └── stage.schema.json
│   │
│   └── proto/                         # Protobuf definitions
│       ├── BUILD.bazel
│       │
│       ├── common/
│       │   ├── BUILD.bazel
│       │   ├── types.proto
│       │   └── status.proto
│       │
│       ├── executor/
│       │   ├── BUILD.bazel
│       │   └── service.proto
│       │
│       ├── controller/
│       │   ├── BUILD.bazel
│       │   └── service.proto
│       │
│       └── compiler/
│           ├── BUILD.bazel
│           └── service.proto
│
├── deploy/                            # Deployment configs
│   ├── BUILD.bazel
│   │
│   ├── kubernetes/
│   │   ├── base/
│   │   │   ├── kustomization.yaml
│   │   │   │
│   │   │   ├── pipeline-executor/
│   │   │   │   ├── deployment.yaml
│   │   │   │   ├── service.yaml
│   │   │   │   ├── configmap.yaml
│   │   │   │   └── hpa.yaml
│   │   │   │
│   │   │   ├── controller/
│   │   │   │   ├── deployment.yaml
│   │   │   │   ├── service.yaml
│   │   │   │   └── configmap.yaml
│   │   │   │
│   │   │   ├── compiler/
│   │   │   │   ├── deployment.yaml
│   │   │   │   └── service.yaml
│   │   │   │
│   │   │   ├── registry/
│   │   │   │   ├── deployment.yaml
│   │   │   │   └── service.yaml
│   │   │   │
│   │   │   ├── ui/
│   │   │   │   ├── deployment.yaml
│   │   │   │   ├── service.yaml
│   │   │   │   └── ingress.yaml
│   │   │   │
│   │   │   └── infrastructure/
│   │   │       ├── kafka.yaml
│   │   │       ├── postgres.yaml
│   │   │       ├── redis.yaml
│   │   │       └── monitoring.yaml
│   │   │
│   │   ├── overlays/
│   │   │   ├── dev/
│   │   │   │   └── kustomization.yaml
│   │   │   ├── staging/
│   │   │   │   └── kustomization.yaml
│   │   │   └── production/
│   │   │       └── kustomization.yaml
│   │   │
│   │   ├── crds/
│   │   │   ├── pipeline.yaml
│   │   │   ├── stage.yaml
│   │   │   └── plugin.yaml
│   │   │
│   │   └── helm/
│   │       └── yeti/
│   │           ├── Chart.yaml
│   │           ├── values.yaml
│   │           ├── values-dev.yaml
│   │           ├── values-prod.yaml
│   │           └── templates/
│   │               ├── deployment.yaml
│   │               ├── service.yaml
│   │               ├── configmap.yaml
│   │               └── ingress.yaml
│   │
│   └── docker/
│       ├── pipeline-executor.Dockerfile
│       ├── controller.Dockerfile
│       ├── compiler.Dockerfile
│       ├── registry.Dockerfile
│       ├── ui.Dockerfile
│       └── cli.Dockerfile
│
├── docs/                              # Documentation
│   ├── architecture/
│   │   ├── overview.md
│   │   ├── components.md
│   │   ├── data-flow.md
│   │   └── decisions/
│   │       ├── 001-plugin-architecture.md
│   │       ├── 002-compiler-optimization.md
│   │       ├── 003-transport-layer.md
│   │       ├── 004-kubernetes-operator.md
│   │       └── 005-hot-reload.md
│   │
│   ├── api/
│   │   ├── rest/
│   │   │   └── api-spec.md
│   │   └── grpc/
│   │       └── services.md
│   │
│   ├── dsl/
│   │   ├── pipeline-schema.md
│   │   ├── examples.md
│   │   └── reference.md
│   │
│   ├── deployment/
│   │   ├── kubernetes/
│   │   │   ├── overview.md
│   │   │   ├── operator.md
│   │   │   └── helm.md
│   │   ├── configuration.md
│   │   └── scaling.md
│   │
│   ├── development/
│   │   ├── setup.md
│   │   ├── building.md
│   │   ├── testing.md
│   │   └── contributing.md
│   │
│   ├── sdk/
│   │   ├── cpp-sdk.md
│   │   ├── go-sdk.md
│   │   ├── python-sdk.md
│   │   └── plugin-development.md
│   │
│   ├── security/
│   │   ├── authentication.md
│   │   ├── authorization.md
│   │   └── encryption.md
│   │
│   └── observability/
│       ├── metrics.md
│       ├── logging.md
│       ├── tracing.md
│       └── dashboards.md
│
├── third_party/                       # External dependencies
│   ├── BUILD.bazel
│   ├── kafka/
│   ├── grpc/
│   ├── prometheus/
│   └── cel/
│
├── tools/                             # Build tools
│   ├── BUILD.bazel
│   │
│   ├── bazel/
│   │   ├── BUILD.bazel
│   │   ├── cpp.bzl
│   │   ├── go.bzl
│   │   └── proto.bzl
│   │
│   ├── codegen/
│   │   ├── BUILD.bazel
│   │   └── protoc_gen_yeti.go
│   │
│   └── scripts/
│       ├── build.sh
│       ├── test.sh
│       ├── deploy.sh
│       └── lint.sh
│
├── tests/                             # Tests
│   ├── BUILD.bazel
│   │
│   ├── unit/
│   │   ├── BUILD.bazel
│   │   ├── executor/
│   │   ├── controller/
│   │   └── compiler/
│   │
│   ├── integration/
│   │   ├── BUILD.bazel
│   │   ├── pipeline_test.cc
│   │   └── stage_test.cc
│   │
│   └── e2e/
│       ├── BUILD.bazel
│       └── scenarios/
│           ├── basic_pipeline_test.go
│           └── complex_pipeline_test.go
│
├── examples/                          # Example pipelines
│   ├── BUILD.bazel
│   ├── basic-pipeline.yaml
│   ├── ecommerce-events.yaml
│   ├── iot-processing.yaml
│   └── custom-plugin/
│       ├── BUILD.bazel
│       └── my_plugin.cc
│
└── scripts/                           # Utility scripts
    ├── setup-dev.sh
    ├── generate-protos.sh
    ├── run-tests.sh
    └── build-all.sh
```

## Ключевые особенности структуры

### 1. Четкое разделение слоев
- **dataplane/**: все компоненты обработки данных (C++)
- **controlplane/**: управляющие сервисы (Go)
- **operator/**: Kubernetes operator (Go)
- **ui/**: веб-интерфейс (TypeScript/React)
- **cli/**: командная утилита (Go)
- **sdk/**: SDKs для расширения

### 2. Плагинная архитектура
```
stages/processors/
├── core/                  # Встроенные плагины
│   ├── filter/
│   ├── map/
│   ├── dedup/
│   └── enrich/
├── loader/                # Динамическая загрузка
└── isolation/             # Режимы изоляции
```

### 3. Transport Layer
```
transport/
├── memory/                # In-process
├── kafka/                 # Distributed
├── grpc/                  # Cross-pod
└── manager/               # Управление
```

### 4. Bazel Build System
- Каждая директория с кодом имеет `BUILD.bazel`
- Инкрементальные сборки
- Кэширование артефактов
- Параллельная компиляция

### 5. Kubernetes-native
```
deploy/kubernetes/
├── base/                  # Базовые манифесты
├── overlays/              # Окружения (dev/staging/prod)
├── crds/                  # Custom Resource Definitions
└── helm/                  # Helm charts
```

Эта структура обеспечивает:
- ✅ Четкую организацию кода
- ✅ Простоту навигации
- ✅ Легкость добавления новых компонентов
- ✅ Эффективную сборку через Bazel
- ✅ Масштабируемость архитектуры
