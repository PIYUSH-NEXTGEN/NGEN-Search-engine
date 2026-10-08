-- name: ListCommunityInfo :many
SELECT id, slug, title, body, created_at, updated_at
FROM community_info
ORDER BY slug;

-- name: UpsertCommunityInfo :one
INSERT INTO community_info (slug, title, body)
VALUES (@slug, @title, @body)
ON CONFLICT (slug) DO UPDATE
  SET title = EXCLUDED.title, body = EXCLUDED.body, updated_at = now()
RETURNING id, slug, title, body, created_at, updated_at;

-- name: GetCommunityInfoBySlug :one
SELECT id, slug, title, body, created_at, updated_at
FROM community_info
WHERE slug = @slug;

-- name: SearchCommunityInfo :many
-- Full-text search across title/body, ranked by relevance.
SELECT
    id,
    slug,
    title,
    body,
    ts_rank(search_vector, plainto_tsquery('english', @query::text)) AS rank
FROM community_info
WHERE search_vector @@ plainto_tsquery('english', @query::text)
ORDER BY rank DESC
LIMIT @result_limit::int;
