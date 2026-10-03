-- name: UpsertTag :one
INSERT INTO tags (name)
VALUES (@name)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING id, name;

-- name: AttachTagToMember :exec
INSERT INTO member_tags (member_id, tag_id)
VALUES (@member_id, @tag_id)
ON CONFLICT DO NOTHING;

-- name: SearchMembersByTag :many
SELECT DISTINCT m.id, m.full_name, m.headline, m.bio, m.location, m.links
FROM members m
JOIN member_tags mt ON mt.member_id = m.id
JOIN tags t ON t.id = mt.tag_id
WHERE t.name = @tag_name AND m.is_public = TRUE
LIMIT @result_limit::int;
