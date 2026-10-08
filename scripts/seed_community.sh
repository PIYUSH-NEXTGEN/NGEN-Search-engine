#!/usr/bin/env bash
# Seeds real community_info rows for Next-Gen Programmers. Safe to run
# repeatedly: community_info rows upsert by slug. psql runs inside the
# compose Postgres container, so no psql install is needed on the host.
# Start the stack first:
#   cd deploy && docker compose up -d
set -euo pipefail

# docker compose resolves the compose file from the working directory.
cd "$(dirname "$0")/../deploy"

# The SQL is piped straight into the container, so the only host requirement
# is a working docker compose — no psql binary needed.
if ! docker compose version >/dev/null 2>&1; then
  echo "error: 'docker compose' is not working in this shell." >&2
  echo "check Docker Desktop is running (and WSL integration if you use WSL)." >&2
  exit 1
fi

if [ -z "$(docker compose ps --status running -q postgres)" ]; then
  echo "error: postgres container is not running." >&2
  echo "start it with: cd deploy && docker compose up -d" >&2
  exit 1
fi

# ON_ERROR_STOP makes psql exit non-zero on any SQL error, so set -e
# actually catches a failed seed instead of printing a false success.
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U community -d community_search <<'SQL'
INSERT INTO community_info (slug, title, body) VALUES
('about', 'About Next-Gen Programmers',
 'Next-Gen Programmers is an online programming community on Discord, founded in July 2025. It is built for programmers to learn, build, collaborate, and grow together, whether they are just starting out or already experienced. The community has been active on Discord since July 2025, with a focus on staying useful and contributor-driven rather than being a server people join and forget.'),
('mission', 'Our Mission',
 'Learn together. Build together. Grow together. Next-Gen Programmers exists to give programmers a place where learning, building, and collaboration happen in the same community, and where members actively contribute instead of just joining.'),
('what-you-can-do', 'What You Can Do Here',
 'Members can ask questions, share their projects, find teammates, exchange resources, and work alongside other developers. Beginners and experienced developers are both welcome.')
ON CONFLICT (slug) DO UPDATE
  SET title = EXCLUDED.title, body = EXCLUDED.body, updated_at = now();
SQL

echo "Seeded community info."