#!/usr/bin/env bash
# Applies (or rolls back) Postgres migrations using golang-migrate.
# Runs the official migrate/migrate image inside the compose network, because
# host-to-Postgres TCP connections fail on this machine — DB host "postgres"
# only resolves inside that network.
#
# Usage:
#   ./scripts/migrate.sh up
#   ./scripts/migrate.sh down 1
#   ./scripts/migrate.sh version
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://community:community@postgres:5432/community_search?sslmode=disable}"
# Resolve to an absolute Windows path (C:/...) so Docker Desktop mounts the
# real folder. A Git-Bash /c/... path would resolve relative to the bash
# install dir instead — that broke the mount with "file://C:/Program
# Files/Git/migrations: no such file or directory".
MIGRATIONS_DIR="$(cd "$(dirname "$0")/../backend/migrations" && pwd -W)"

# MSYS2/Git-Bash rewrites "C:/..."-looking args to POSIX paths before docker
# sees them (turning the mount into "C:/Program Files/Git/migrations").
# Excluding everything from conversion is the reliable fix — this script
# passes nothing that needs MSYS path rewriting.
export MSYS2_ARG_CONV_EXCL="*"
docker run --rm --network deploy_default \
  -v "${MIGRATIONS_DIR}:/migrations" \
  migrate/migrate -path=/migrations -database "$DATABASE_URL" "$@"
