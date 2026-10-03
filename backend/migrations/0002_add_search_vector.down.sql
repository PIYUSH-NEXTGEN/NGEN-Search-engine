DROP INDEX IF EXISTS members_search_idx;
ALTER TABLE members DROP COLUMN IF EXISTS search_vector;
