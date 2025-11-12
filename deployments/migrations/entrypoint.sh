#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-admin}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-password}"
POSTGRES_DBNAME="${POSTGRES_DBNAME:-filtering}"
POSTGRES_SSLMODE="${POSTGRES_SSLMODE:-disable}"

MONGODB_HOST="${MONGODB_HOST:-mongodb}"
MONGODB_PORT="${MONGODB_PORT:-27017}"
MONGODB_USER="${MONGODB_USER:-admin}"
MONGODB_PASSWORD="${MONGODB_PASSWORD:-password}"
MONGODB_DATABASE="${MONGODB_DATABASE:-enrichment}"
MONGODB_AUTH_SOURCE="${MONGODB_AUTH_SOURCE:-admin}"

MIGRATIONS_PATH="${MIGRATIONS_PATH:-/migrations}"
POSTGRES_MIGRATIONS_PATH="${POSTGRES_MIGRATIONS_PATH:-${MIGRATIONS_PATH}/postgres}"
MONGODB_MIGRATIONS_PATH="${MONGODB_MIGRATIONS_PATH:-${MIGRATIONS_PATH}/mongodb}"

echo "=== Migration Service ==="
echo "PostgreSQL: ${POSTGRES_USER}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DBNAME}"
echo "MongoDB: ${MONGODB_USER}@${MONGODB_HOST}:${MONGODB_PORT}/${MONGODB_DATABASE}"
echo "Migrations path: ${MIGRATIONS_PATH}"
echo ""

echo "Waiting for PostgreSQL to be ready..."
until pg_isready -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}"; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 1
done
echo "PostgreSQL is ready!"

echo "Waiting for MongoDB to be ready..."
until mongosh --host "${MONGODB_HOST}" --port "${MONGODB_PORT}" \
    --username "${MONGODB_USER}" --password "${MONGODB_PASSWORD}" \
    --authenticationDatabase "${MONGODB_AUTH_SOURCE}" \
    --eval "db.adminCommand('ping')" --quiet > /dev/null 2>&1; do
    echo "MongoDB is unavailable - sleeping"
    sleep 1
done
echo "MongoDB is ready!"
echo ""

if [ -d "${POSTGRES_MIGRATIONS_PATH}" ] && [ -n "$(find ${POSTGRES_MIGRATIONS_PATH} -name '*.sql' -type f 2>/dev/null)" ]; then
    echo "=== Running PostgreSQL migrations ==="
    POSTGRES_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DBNAME}?sslmode=${POSTGRES_SSLMODE}"
    
    migrate -path="${POSTGRES_MIGRATIONS_PATH}" -database="${POSTGRES_URL}" up
    MIGRATE_EXIT_CODE=$?

    if [ $MIGRATE_EXIT_CODE -eq 0 ]; then
        echo "PostgreSQL migrations completed successfully"
    elif [ $MIGRATE_EXIT_CODE -eq 1 ]; then
        echo "PostgreSQL migrations: no change (already up to date)"
    else
        echo "PostgreSQL migrations failed with exit code: $MIGRATE_EXIT_CODE"
        exit $MIGRATE_EXIT_CODE
    fi
    echo ""
else
    echo "No PostgreSQL migrations found in ${POSTGRES_MIGRATIONS_PATH}, skipping..."
    echo ""
fi

if [ -d "${MONGODB_MIGRATIONS_PATH}" ] && [ -n "$(find ${MONGODB_MIGRATIONS_PATH} -name '*.js' -type f 2>/dev/null)" ]; then
    echo "=== Running MongoDB migrations ==="
    MONGODB_URL="mongodb://${MONGODB_USER}:${MONGODB_PASSWORD}@${MONGODB_HOST}:${MONGODB_PORT}/${MONGODB_DATABASE}?authSource=${MONGODB_AUTH_SOURCE}"

    for migration_file in $(find ${MONGODB_MIGRATIONS_PATH} -name '*.js' -type f | sort); do
        migration_name=$(basename "$migration_file")
        echo "Running migration: $migration_name"

        MIGRATION_CHECK=$(mongosh "${MONGODB_URL}" --quiet --eval "db.migrations.findOne({name: '${migration_name}'})" 2>/dev/null || echo "")
        
        if [ -n "$MIGRATION_CHECK" ] && [ "$MIGRATION_CHECK" != "null" ]; then
            echo "Migration $migration_name already applied, skipping..."
            continue
        fi

        if mongosh "${MONGODB_URL}" --file "$migration_file" --quiet; then
            mongosh "${MONGODB_URL}" --quiet --eval "db.migrations.insertOne({name: '${migration_name}', applied_at: new Date()})" > /dev/null 2>&1 || true
            echo "Migration $migration_name completed successfully"
        else
            MONGOSH_EXIT_CODE=$?
            echo "Migration $migration_name failed with exit code: $MONGOSH_EXIT_CODE"
            exit $MONGOSH_EXIT_CODE
        fi
    done
    echo "MongoDB migrations completed successfully"
    echo ""
else
    echo "No MongoDB migrations found in ${MONGODB_MIGRATIONS_PATH}, skipping..."
    echo ""
fi

echo "=== All migrations completed successfully ==="
