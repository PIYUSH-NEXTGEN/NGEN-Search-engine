#!/usr/bin/env bash
# Regenerates internal/store/queries/*.sql.go from internal/store/sql/*.sql
# using sqlc. Install: https://docs.sqlc.dev/en/latest/overview/install.html
set -euo pipefail
cd "$(dirname "$0")/../backend"
sqlc generate
echo "sqlc generate complete."
