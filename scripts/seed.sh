#!/usr/bin/env bash
# Loads a handful of sample members so you have something to search
# against locally. Adjust DATABASE_URL if not using the default compose setup.
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://community:community@localhost:5432/community_search?sslmode=disable}"

psql "$DATABASE_URL" <<'SQL'
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
