# Community search engine

Phase 0/1 scaffold: Go backend with Postgres full-text search + Redis caching,
and a minimal Next.js/TypeScript frontend. No LLM layer yet (that's Phase 2 —
see `internal/llm/` as the next package to fill in).

## Run it locally

### 1. Start Postgres + Redis + backend via Docker Compose
```bash
cd deploy
docker compose up -d --build
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

### 4. Try the API directly
```bash
curl "http://localhost:8080/api/search?q=machine+learning"
```

### 5. Run the frontend
```bash
cd frontend
cp ../.env.example .env.local   # then adjust NEXT_PUBLIC_API_BASE_URL if needed
npm install
npm run dev
```
Visit http://localhost:3000, search for "machine learning" — you should see
Asha Rao come back from the seeded data.

## Project layout
See `project-structure.md` (or the earlier build guide) for the full
folder-by-folder explanation. Quick map:
- `backend/cmd/server` — entrypoint, wires everything together
- `backend/internal/search` — retrieval orchestration (cache + DB)
- `backend/internal/store` — Postgres access; `sql/` is hand-written,
  `queries/` is sqlc-generated (checked in by hand for now — see below)
- `backend/internal/cache` — Redis: search cache, rate limiting
- `backend/internal/api` — HTTP handlers and router
- `backend/migrations` — versioned schema changes
- `frontend/` — Next.js app

## About the sqlc-generated files
`backend/internal/store/queries/*.sql.go` are written by hand in this
scaffold to match exactly what `sqlc generate` would produce, so the project
builds without you needing sqlc installed yet. Once you install
[sqlc](https://docs.sqlc.dev/en/latest/overview/install.html), running
`./scripts/gen.sh` will regenerate them from `backend/internal/store/sql/*.sql`
— review the diff once to confirm it matches, then treat that as the source
of truth going forward and stop hand-editing.

## Next step: Phase 2 (LLM answer layer)
Not scaffolded yet. When ready: add `backend/internal/llm/{client,prompt,types}.go`,
an `/api/ask` handler that calls `search.Service` then the LLM, and an
`AnswerPanel.tsx` component on the frontend. Ask for this scaffold whenever
you're ready to build it.

## Environment variables
See `.env.example` at the repo root.
