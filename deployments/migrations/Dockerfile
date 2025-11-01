FROM mongo:6

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl \
    postgresql-client \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

ARG MIGRATE_VERSION=4.17.0
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz | \
    tar -xz -C /tmp && \
    mv /tmp/migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

WORKDIR /migrations

COPY deployments/migrations/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
