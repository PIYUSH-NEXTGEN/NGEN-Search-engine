#!/usr/bin/env bash
# Loads a handful of sample members so you have something to search
# against locally. psql runs inside the compose Postgres container, so no
# psql install is needed on the host. Start the stack first:
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
INSERT INTO members (full_name, headline, bio, location, is_public) VALUES
('Asha Rao', 'ML engineer, ex-Google', 'Asha builds recommendation systems and has spent the last five years working on large-scale machine learning infrastructure. Previously at Google Brain.', 'Bengaluru', TRUE),
('Marcus Webb', 'Fintech founder', 'Marcus co-founded a payments startup focused on cross-border remittances in Southeast Asia. Background in distributed systems.', 'Singapore', TRUE),
('Priya Nair', 'Data scientist', 'Priya works on causal inference and experimentation platforms, with a research background in statistics.', 'Mumbai', TRUE);

INSERT INTO tags (name) VALUES ('machine-learning'), ('fintech'), ('data-science') ON CONFLICT DO NOTHING;

INSERT INTO member_tags (member_id, tag_id)
SELECT m.id, t.id FROM members m, tags t
WHERE m.full_name = 'Asha Rao' AND t.name = 'machine-learning'
ON CONFLICT DO NOTHING;

INSERT INTO member_tags (member_id, tag_id)
SELECT m.id, t.id FROM members m, tags t
WHERE m.full_name = 'Marcus Webb' AND t.name = 'fintech'
ON CONFLICT DO NOTHING;

INSERT INTO member_tags (member_id, tag_id)
SELECT m.id, t.id FROM members m, tags t
WHERE m.full_name = 'Priya Nair' AND t.name = 'data-science'
ON CONFLICT DO NOTHING;
SQL

echo "Seeded sample members."
