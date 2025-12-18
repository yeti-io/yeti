#!/usr/bin/env bash

set -e

echo "Generating Swagger documentation..."

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SWAG_BIN="$(go env GOPATH)/bin/swag"

if [ ! -f "$SWAG_BIN" ]; then
    echo "Installing swag..."
    go install github.com/swaggo/swag/cmd/swag@latest
fi

cd "$PROJECT_ROOT/cmd/management-service"

"$SWAG_BIN" init -g main.go -o docs --parseDependency --parseInternal

echo "Swagger documentation generated successfully!"
echo "Documentation is available at: http://localhost:8080/swagger/index.html"
echo "Generated files:"
echo "  - docs/docs.go"
echo "  - docs/swagger.json"
echo "  - docs/swagger.yaml"
