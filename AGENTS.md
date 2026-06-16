# Agent Rules — Godploy

## Project Context

**Godploy** is a lightweight, single-binary, self-hosted PaaS (Platform as a Service) — an alternative to Dokploy and Coolify.
**Stack:** Go (Echo) · SvelteKit SPA (embedded in binary) · SQLite (via sqlc) · Docker · Traefik

To understand the project, read:
- **PRD:** `./docs/prd.md`
- **Context:** `./docs/CONTEXT.md`

**Structure:**

- `backend/` — production-grade Go backend following clean architecture
  - `cmd/` — entrypoints
    - `server/` — HTTP server main
    - `setup/` — CLI setup commands
    - `sample/` — sample data generation
  - `internal/` — app logic organized by concern
    - `config/` — configuration loaders
    - `db/` — sqlc-generated database layer (models, queries) — **do not modify**; generated via `make generate` from `sqlite/query/*.sql`
    - `handlers/` — HTTP handlers (auth, project, service, github, health)
    - `jobs/` — background job processing
    - `lib/` — utilities (session, password, csrf, docker, github install)
    - `middleware/` — HTTP middleware (auth, cors, rate limiting)
    - `routes/` — route registration
    - `service/` — business logic layer
  - `sqlite/` — migrations and raw SQL queries (sqlc input)
  - `frontend/` — embedded SvelteKit SPA build output
  - `integration_tests/` — integration test suites
- `frontend/` — SvelteKit SPA source (see `frontend/AGENTS.md`)
- `docs/` — project documentation

---

## Code Rules

### Comments

- **Standard operations** (API handlers, DB queries, route setup) — no comments needed. These are familiar territory.
- **New patterns, utility functions, unfamiliar abstractions** — add a short, crisp comment above them explaining _what_ and _why_. One or two lines max.
- **AI-generated code** — always add a brief summary comment explaining the design, concept, or reasoning behind the implementation. This helps document the thought process behind new patterns or approaches.
- Comments should be straight to the point. No fluff.

### Knowledge Capture

- When introducing a new design, concept, or engineering approach in code, update `/docs/queries.md` with the topic
- Include a brief explanation or reference that captures the core idea
- This ensures new learnings are captured and searchable in one place

### Style

- Follow existing conventions in the codebase (formatting, naming, structure)
- Don't introduce new libraries or patterns without the owner understanding why
- Keep changes minimal — only touch what's needed

---

## Documentation Rules

- The owner is a learner — write docs in **simple language but keep them fully technical**
- Don't oversimplify to the point of losing accuracy
- Don't over-explain to the point of being patronizing
- Reference existing docs and code rather than duplicating information
