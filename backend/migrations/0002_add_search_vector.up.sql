ALTER TABLE members ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(full_name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(headline, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(bio, '')), 'C')
  ) STORED;

CREATE INDEX members_search_idx ON members USING GIN (search_vector);
