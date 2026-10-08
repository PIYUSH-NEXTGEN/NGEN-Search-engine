-- name: CreateProject :one
INSERT INTO projects (name, description, status, link)
VALUES (@name, @description, @status, @link)
RETURNING id, name, description, status, link, created_at, updated_at;

-- name: GetProjectByID :one
SELECT id, name, description, status, link, created_at, updated_at
FROM projects
WHERE id = @id;

-- name: SearchProjects :many
-- Full-text search across name/description, ranked by relevance.
SELECT
    id,
    name,
    description,
    status,
    link,
    ts_rank(search_vector, plainto_tsquery('english', @query::text)) AS rank
FROM projects
WHERE search_vector @@ plainto_tsquery('english', @query::text)
ORDER BY rank DESC
LIMIT @result_limit::int;

-- name: AttachMemberToProject :exec
INSERT INTO project_members (project_id, member_id)
VALUES (@project_id, @member_id)
ON CONFLICT DO NOTHING;

-- name: AttachTagToProject :exec
INSERT INTO project_tags (project_id, tag_id)
VALUES (@project_id, @tag_id)
ON CONFLICT DO NOTHING;

-- name: ListProjectMembers :many
SELECT m.id, m.full_name
FROM members m
JOIN project_members pm ON pm.member_id = m.id
WHERE pm.project_id = @project_id AND m.is_public = TRUE;

-- name: ListProjectTags :many
SELECT t.id, t.name
FROM tags t
JOIN project_tags pt ON pt.tag_id = t.id
WHERE pt.project_id = @project_id;
