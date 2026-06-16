# ADR 0001: Service Dependency Graph with Dynamic Deploy-Time Resolution

## Status

Accepted

## Context

Godploy manages multiple service types (application services, PostgreSQL, Redis) within isolated **Project Instances**. Services within an instance communicate via **Internal URLs** over the **Instance Network**. Currently, connecting an application service to a database requires the user to manually copy the database's `internal_url` and paste it into the application's environment variables. This manual process:

- Breaks during preview instance creation, because the user must manually update every environment variable to point to the preview-cloned service equivalents
- Leaves no traceable metadata about which service depends on which, preventing the UI from showing a connection graph
- Prevents future features like "deploy only services affected by this PR" from knowing the dependency topology

The product needs an explicit, optional dependency declaration mechanism that:

1. Lets users connect services via UI without manual copy-paste
2. Automatically rewrites connections during preview cloning
3. Remains fully optional so existing manual env configuration continues to work

## Decision

We will implement a **Service Dependency Graph** using **dynamic deploy-time resolution** (Option B from the design discussion):

- A `service_dependencies` table stores: `app_service_id` (FK), `dependency_service_id` (plain text, no FK across service type tables), `dependency_service_type`, `dependency_column`, `env_var_name`, `env_value`
- The `env_value` is a **cached snapshot** of the dependency's column value at connection time, updated by a background routine when the dependency changes
- At deploy time, the deploy job queries all dependency records for the service, resolves the current value from the dependency's table (or uses the cached `env_value`), and merges them into the runtime environment
- **Dependency-derived env vars take precedence** over user-defined env vars on conflict
- During preview instance creation, dependency records are cloned and `dependency_service_id` is rewritten to point to the preview-cloned service equivalents
- The feature is **fully optional** — users who don't use it continue with manual env configuration

## Consequences

### Positive

- Preview instances automatically get correct dependency connections without user intervention
- UI can render a service connection graph from the dependency table
- Background update routine keeps dependency values current when credentials change
- Deploy-time resolution means the system always uses the latest value (with cache fallback)
- Optional adoption means no breaking change for existing users

### Negative

- The `service_dependencies` table has no foreign key to `dependency_service_id` because it can reference `app_service`, `psql_service`, or `redis_service`. This requires deploy-time logic to query the correct table based on `dependency_service_type`.
- Cached `env_value` can become stale if the background update routine fails or is delayed. The deploy-time resolver should prefer live resolution and use the cache only as fallback.
- Dependency records must be manually cleaned up when a service is deleted (no cascade delete via FK).
- Adding a new service type in the future requires updating the deploy-time resolver to know about the new table.

### Neutral

- The decision explicitly rejects Railway-style template syntax (`${{ service.VAR }}`) in favor of a UI-driven connect flow, trading power-user flexibility for discoverability.
- The decision explicitly rejects auto-cascading redeploy of dependent services when a dependency changes; dependents pick up new values on their next deploy (manual, webhook-triggered, or preview creation).
