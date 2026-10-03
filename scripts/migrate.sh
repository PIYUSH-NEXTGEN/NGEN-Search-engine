#!/usr/bin/env bash
# Applies (or rolls back) Postgres migrations using golang-migrate.
# Install: https://github.com/golang-migrate/migrate
#
# Usage:
#   ./scripts/migrate.sh up
#   ./scripts/migrate.sh down 1
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://community:community@localhost:5432/community_search?sslmode=disable}"
MIGRATIONS_DIR="$(dirname "$0")/../backend/migrations"

migrate -database "$DATABASE_URL" -path "$MIGRATIONS_DIR" "$@"
