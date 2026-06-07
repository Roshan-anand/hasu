**Frontend**

This document describes the frontend stack, preferred tooling, and available agent skills for the project.

**Structure:**

- `src/` — application source
  - `lib/` — shared code
    - `features/` — domain-organized modules (`auth/`, `base/`, `deployments/`, `git/`, `global/`, `instance/`, `services/`); each domain houses its own **queries, mutations, types, contexts**, and related logic
    - `components/` — shared UI components
    - `hooks/` — shared Svelte runes/hooks
    - `axios.ts` — HTTP client config
    - `query.ts` — TanStack Query client setup
    - `types.ts` — shared types
    - `utils.ts` — shared utilities
    - `assets/` — static assets
  - `routes/` — SvelteKit routes
  - `tests/` — test suites
- `static/` — static assets
- `svelte.config.ts` — SvelteKit configuration
- `vite.config.ts` — Vite bundler configuration
- `components.json` — shadcn component registry
- `package.json` — dependencies and scripts

**Stack**

- Svelte 5
- SvelteKit
- TanStack Query (for server-state and caching)
- shadcn (design system / component primitives)
- Other major libraries and packages used as needed (routing, form libraries, CSS utilities)

**Package Manager**

- Bun is the canonical package manager for the frontend. Use `bun` and `bunx` commands for installs and scripts.

**Component Library**

- Use shadcn for UI components and primitives. When a required component is not present in the codebase, install it with the shadcn Svelte installer. Example:

```bash
bun x shadcn-svelte@latest add button
```

**Available Skills**

- shadcn
- tanstack-query
- tanstack-form
- svelte-code-writer
- svelte-core-bestpractices

**Usage Notes**

- Prefer shadcn components for UI consistency; only add new components when needed.
- Use TanStack Query for server-state, optimistic updates, and efficient caching patterns.
- Use Tanstack Form for form state management, validation, and submission handling.
- Follow the svelte-core-bestpractices skill for modern Svelte patterns and performance guidance.

IMPORTANT (Global rule):

- When writing API handlers, DB queries, UI components, query/mutation utilities, or types, follow the existing patterns in the codebase. Use nearby or similar examples as the canonical style and structure to keep code consistent.
