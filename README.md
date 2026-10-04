# Community search engine

Go backend (Postgres full-text search + Redis caching + Gemini answer
layer with session follow-ups) and a Next.js/TypeScript frontend that shows
the generated answer above member cards with a follow-up thread.

## Run it locally

### 1. Start Postgres + Redis + backend via Docker Compose
```bash
cd deploy
docker compose up -d --build
```

The backend reads its config from `deploy/.env` (Compose interpolates
`${...}` from that file automatically — no `env_file:` needed). At minimum
it needs a Gemini key:

```bash
LLM_API_KEY=<key from https://aistudio.google.com/apikey>
LLM_MODEL=gemini-3.8-flash
```

### 2. Run migrations
Install [golang-migrate](https://github.com/golang-migrate/migrate) first, then:
```bash
DATABASE_URL="postgres://community:community@localhost:5432/community_search?sslmode=disable" \
  ./scripts/migrate.sh up
```

### 3. Seed sample data
```bash
./scripts/seed.sh
```
This inserts 3 sample members. Delete them afterward if you use real data:
```sql
BEGIN;
DELETE FROM member_tags WHERE member_id IN (SELECT id FROM members WHERE full_name IN ('Asha Rao','Marcus Webb','Priya Nair'));
DELETE FROM members WHERE full_name IN ('Asha Rao','Marcus Webb','Priya Nair');
DELETE FROM links WHERE tag_id IN (SELECT id FROM tags WHERE name IN ('recommendation systems','distributed systems','design systems'));
DELETE FROM tags WHERE name IN ('recommendation systems','distributed systems','design systems');
COMMIT;
```

### 4. Try the API directly
```bash
curl "http://localhost:8080/api/search?q=machine+learning"
curl -X POST http://localhost:8080/api/ask \
  -H 'Content-Type: application/json' \
  -d '{"query":"machine learning"}'
# Follow-up: reuse the session_id from the response above.
curl -X POST http://localhost:8080/api/ask \
  -H 'Content-Type: application/json' \
  -d '{"query":"where are they based?","session_id":"<that id>"}'
```

### 5. Run the frontend
```bash
cd frontend
cp ../.env.example .env.local   # then adjust NEXT_PUBLIC_API_BASE_URL if needed
npm install
npm run dev
```
Visit http://localhost:3000, search for "machine learning" — you should see
a generated paragraph above the member cards, plus a follow-up box that
keeps prior turns visible in a thread.

## Project layout
Quick map:
- `backend/cmd/server` — entrypoint, wires everything together
- `backend/internal/search` — retrieval orchestration (cache + DB)
- `backend/internal/store` — Postgres access; `sql/` is hand-written,
  `queries/` is sqlc-generated (checked in by hand for now — see below)
- `backend/internal/cache` — Redis: search cache, answer cache, rate limiting
- `backend/internal/api` — HTTP handlers and router (`/api/search`, `/api/ask`)
- `backend/internal/llm` — Gemini answer layer: prompt builder, client, types
- `backend/internal/session` — Redis-backed follow-up sessions (`session:<id>`, 30 min TTL)
- `backend/migrations` — versioned schema changes
- `frontend/` — Next.js app (`AnswerPanel` + `FollowUpInput` on the search page)

## How /api/ask works
`POST /api/ask {"query", "session_id?"}` runs `search.Service` first, then
asks Gemini (`gemini-3.8-flash` by default) for `{"relevant", "answer"}` —
grounded strictly in the retrieved records plus prior turns from the
caller's Redis session. Context-free answers are cached 30 min; follow-ups
with history bypass the cache. The response includes the raw `results` plus
a `session_id` the frontend keeps in React state for the thread.

## About the sqlc-generated files
`backend/internal/store/queries/*.sql.go` are written by hand in this
scaffold to match exactly what `sqlc generate` would produce, so the project
builds without you needing sqlc installed yet. Once you install
[sqlc](https://docs.sqlc.dev/en/latest/overview/install.html), running
`./scripts/gen.sh` will regenerate them from `backend/internal/store/sql/*.sql`
— review the diff once to confirm it matches, then treat that as the source
of truth going forward and stop hand-editing.

## Environment variables
See `.env.example` at the repo root. The backend never reads a `.env` file
itself — Compose interpolates `deploy/.env` into the container environment.
