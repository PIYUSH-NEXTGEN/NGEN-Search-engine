-- name: SearchMembers :many
-- Full-text search across name/headline/bio, ranked by relevance,
-- plus a boost for members whose tags match the raw query text.
SELECT
    m.id,
    m.full_name,
    m.headline,
    m.bio,
    m.location,
    m.links,
    ts_rank(m.search_vector, plainto_tsquery('english', @query::text)) AS rank
FROM members m
WHERE m.is_public = TRUE
  AND m.search_vector @@ plainto_tsquery('english', @query::text)
ORDER BY rank DESC
LIMIT @result_limit::int;

-- name: GetMemberByID :one
SELECT id, full_name, headline, bio, location, links, is_public, created_at, updated_at
FROM members
WHERE id = @id;

-- name: CreateMember :one
INSERT INTO members (full_name, headline, bio, location, email, links, is_public)
VALUES (@full_name, @headline, @bio, @location, @email, @links, @is_public)
RETURNING id, full_name, headline, bio, location, links, is_public, created_at, updated_at;

-- name: ListMemberTags :many
SELECT t.name
FROM tags t
JOIN member_tags mt ON mt.tag_id = t.id
WHERE mt.member_id = @member_id;
