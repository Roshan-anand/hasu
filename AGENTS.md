# Agent Rules — Godploy

## Project Context

**Godploy** is a lightweight, single-binary, self-hosted PaaS (Platform as a Service) — an alternative to Dokploy and Coolify.
**Stack:** Go (Echo) · SvelteKit SPA (embedded in binary) · SQLite (via sqlc) · Docker · Traefik
**Monorepo:** Bun workspaces + Turborepo

To understand the project, read:
- **PRD:** `./docs/prd.md`
- **Context:** `./docs/CONTEXT.md`

**Structure:**

```
godploy/
├── apps/
│   ├── server/            — Go backend (Echo v5, SQLite, Docker SDK)
│   │   ├── cmd/           — entrypoints (server, setup, sample)
│   │   ├── internal/      — app logic
│   │   │   ├── config/    — configuration loaders
│   │   │   ├── db/        — sqlc-generated database layer — **do not modify**; generated via `make generate`
│   │   │   ├── handlers/  — HTTP handlers (auth, project, service, github, health)
│   │   │   ├── jobs/      — background job processing
│   │   │   ├── lib/       — utilities (session, password, csrf, docker, github install)
│   │   │   ├── middleware/ — HTTP middleware (auth, cors, rate limiting)
│   │   │   ├── routes/    — route registration
│   │   │   └── service/   — business logic layer
│   │   ├── sqlite/        — migrations and raw SQL queries (sqlc input)
│   │   ├── frontend/      — embedded SvelteKit SPA build output (from apps/web)
│   │   └── integration_tests/
│   └── web/               — SvelteKit SPA frontend (see apps/web/AGENTS.md)
├── packages/              — reserved for shared packages (currently empty)
├── docker/                — Docker Compose files (dev, Traefik)
└── docs/                  — project documentation
```

**Key configs:**
- `package.json` — Bun workspaces (`apps/*`, `packages/*`)
- `turbo.json` — Turborepo pipeline (build, dev, check-types, lint, format)
- `makefile` — dev convenience commands (`make start`, `make build`, `make test`, etc.)

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
