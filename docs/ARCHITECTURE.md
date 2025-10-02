# ETL Platform Architecture

This document provides a comprehensive overview of the ETL Platform architecture, design decisions, and implementation details.

## Table of Contents

- [Overview](#overview)

## Overview

## Core Components

| Component              | Purpose                       | Technology |
|------------------------|-------------------------------|------------|
| **Pipeline Manager**   | Pipeline lifecycle management | Go, gRPC |
| **Processing Engine**  | ETL execution engine          | Go, custom DAG engine |
| **Config Service**     | Configuration management      | Go, gRPC |
| **Monitoring Service** | Metrics and observability     | Go, gRPC, Prometheus |
| **Loading Engine**     | Stream data loading           | Go, Kafka |
| **CLI Tool (yetictl)** | Command-line interface        | Go, Cobra |
| **Web UI**             | Web interface                 | React/TypeScript |

## Technology Stack

### Backend Services
- **Language**: Go
- **API**: gRPC with gRPC-Gateway (REST)
- **Framework**: Standard library + specialized libraries
- **Database**: PostgreSQL
- **Cache**: Redis
- **Message Queue**: Apache Kafka
- **Storage**: S3-compatible (MinIO/Ceph)

### Frontend
ХЗ

### Infrastructure
- **Container Runtime**: Containerd
- **Orchestration**: Kubernetes
- **Monitoring**: Prometheus + Grafana or Signoz
- **Tracing**: Jaeger or Signoz
- **Secrets**: Vault or Kubernetes Secrets

---

**Last Updated**: 2025-09-30  
**Version**: 0.0.1
**Maintainers**: Zherdev Egor