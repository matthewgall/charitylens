#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/docker-compose.smoke.yml"

cleanup() {
  docker compose -f "${COMPOSE_FILE}" down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose -f "${COMPOSE_FILE}" up -d

echo "Running PostgreSQL API smoke test..."
SMOKE_DB=postgres \
SMOKE_DSN="host=127.0.0.1 port=55432 user=charitylens password=charitylens dbname=charitylens sslmode=disable" \
go test ./internal/handlers -tags=integration -run TestAPIEndpointsSmoke -count=1

echo "Running MySQL API smoke test..."
SMOKE_DB=mysql \
SMOKE_DSN="charitylens:charitylens@tcp(127.0.0.1:53306)/charitylens?parseTime=true" \
go test ./internal/handlers -tags=integration -run TestAPIEndpointsSmoke -count=1

echo "Smoke tests passed for PostgreSQL and MySQL."
