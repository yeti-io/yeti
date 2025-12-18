FROM debian:12-slim as builder

ARG JMX_EXPORTER_VERSION=1.4.0

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt/jmx-exporter && \
    curl -sSL \
    "https://github.com/prometheus/jmx_exporter/releases/download/${JMX_EXPORTER_VERSION}/jmx_prometheus_standalone-${JMX_EXPORTER_VERSION}.jar" \
    -o /opt/jmx-exporter/jmx_prometheus_standalone.jar && \
    chmod 644 /opt/jmx-exporter/jmx_prometheus_standalone.jar

FROM eclipse-temurin:21-jre-alpine

LABEL maintainer="your-email@example.com"
LABEL org.opencontainers.image.description="JMX Exporter for Prometheus"

RUN apk add --no-cache wget

COPY --from=builder /opt/jmx-exporter /opt/jmx-exporter

RUN addgroup -S jmxgroup && \
    adduser -S -G jmxgroup -u 1001 jmxuser && \
    chown -R jmxuser:jmxgroup /opt/jmx-exporter

USER jmxuser

WORKDIR /etc/jmx-exporter

EXPOSE 5556

ENTRYPOINT ["java", "-jar", "/opt/jmx-exporter/jmx_prometheus_standalone.jar"]
CMD ["5556", "/etc/jmx-exporter/config.yml"]
