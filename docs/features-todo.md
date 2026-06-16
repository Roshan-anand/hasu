# Service Dependency Graph — PRD

## Problem Statement

User creates multi-service project (backend + Postgres + Redis). To connect them, user must manually copy each service's **Internal URL**, paste into each dependent service's env vars. In production — tedious but works once. In preview — breaks completely. Every preview clone needs manual env rewrite to point at cloned services. No visibility into which service connects to which. No automation for preview injection.

## Solution

Optional **Service Dependency Graph**. User explicitly declares connections between services via UI. System stores dependency records, resolves values at deploy time, rewrites connections automatically for previews. Manual env config still works for users who don't opt in.

## User Stories

1. As a logged-in user, I want to connect an application Service to a Predefined Database Service (Postgres/Redis), so that I don't need to manually copy-paste connection strings into env vars.
2. As a logged-in user, I want to connect an application Service to another application Service, so that internal service-to-service wiring is declared and managed by the platform.
3. As a logged-in user, I want to see a "Connect Service" option in each application Service's settings page, so that I can discover and use the dependency feature.
4. As a logged-in user, I want the connect UI to show a dropdown of all other running Services in the same Project Instance, so that I can pick the target service visually.
5. As a logged-in user, I want to specify the environment variable name (e.g., `DATABASE_URL`) when creating a connection, so that my application receives the value under the expected name.
6. As a logged-in user, I want to select which column from the target service to inject (e.g., `internal_url`, `db_name`, `db_password`), so that I have fine-grained control over what value my app receives.
7. As a logged-in user, I want to add multiple connections from one application Service (e.g., `DATABASE_URL` + `REDIS_URL`), so that one app can depend on multiple services.
8. As a logged-in user, I want the system to show a toast reminder "Don't forget to redeploy" after adding or changing a connection, so that I know the connection only takes effect on next deploy.
9. As a logged-in user, I want dependency-derived env vars to take precedence over manually-set env vars of the same name, so that the managed connection always wins on conflict.
10. As a logged-in user, I want manual env var configuration to continue working unchanged, so that I can choose not to use the dependency feature.
11. As a logged-in user, I want to view the list of all connections for a Service in its settings page, so that I can audit and manage the dependency graph.
12. As a logged-in user, I want to delete an individual connection, so that I can remove a dependency that is no longer needed.
13. As a logged-in user, I want to edit an existing connection's env var name or target column, so that I can correct or update the wiring.
14. As a developer creating a preview Project Instance, I want all Service Dependencies to be automatically rewritten to point to the preview-cloned service equivalents, so that preview services connect correctly without manual intervention.
15. As a developer creating a preview, I want dependency-derived env vars injected into preview services at deploy time just like production, so that previews are fully functional with zero manual env editing.
16. As a logged-in user, I want to see a visual dependency graph in the UI showing which services connect to which, so that I can understand the instance topology at a glance.
17. As a logged-in user, I want the dependency graph to remain correct when I create a new preview — showing the cloned services and their connections — so that preview topology is visible.
18. As a user who changes Postgres credentials, I want my connected application services to receive the updated connection string on their next deploy, so that I don't need to manually update env vars after a credential rotation.
19. As a logged-in user, I want the system to handle service deletion gracefully — deleting all dependency records associated with the deleted service, so that no dangling references remain.
20. As a logged-in user, I want to see in the service settings which connections have stale values (dependency changed since last deploy), so that I know when a redeploy would pick up new values.

## Schema

```sql
CREATE TABLE service_dependencies (
    id                        uuid PRIMARY KEY,
    app_service_id            uuid NOT NULL,
    dependency_service_id     TEXT NOT NULL,
    dependency_service_type   TEXT NOT NULL,   -- 'app', 'psql', 'redis'
    dependency_column         TEXT NOT NULL,   -- e.g., 'internal_url', 'db_name', 'db_password', 'domain', 'name'
    env_var_name              TEXT NOT NULL,
    env_value                 TEXT NOT NULL,   -- cached snapshot, updated by bg routine
    created_at                DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    UNIQUE(app_service_id, env_var_name)
);
```

**No FK** on `dependency_service_id` — references `app_service.id`, `psql_service.id`, or `redis_service.id`. Resolution uses `dependency_service_type` to pick correct table.

**No FK** on `app_service_id` — cascading delete handled manually at service deletion.

## Available Columns per Service Type

| Service Type | Connectable Columns |
|---|---|
| `app` | `internal_url`, `domain` |
| `psql` | `internal_url`, `db_name`, `db_user`, `db_password` |
| `redis` | `internal_url`, `password`, `name` |

## UX Flow

### Connect Service
1. User navigates to application Service settings → Environment variables section
2. Clicks **"Connect Service"** button
3. Modal/drawer opens with:
   - **Input**: env var name (e.g., `DATABASE_URL`)
   - **Select**: target service (dropdown of all Services in same Project Instance)
   - **Select**: target column (dropdown of available columns for that service type)
   - **"Add"** button — adds row to table below the inputs
4. User can add multiple connections (one row each)
5. User clicks **"Save"** → server creates `service_dependencies` records → shows toast "Don't forget to redeploy"
6. New connections appear in env vars list with visual indicator showing they're dependency-managed

### Edit Connection
1. User clicks row in connections list
2. Same form reopens pre-filled → user changes env var name or target column
3. Save → server updates record → toast "Don't forget to redeploy"

### Delete Connection
1. User clicks delete icon on connection row
2. Confirmation dialog
3. Delete → server removes record → toast "Don't forget to redeploy"

## Deploy-Time Resolution

```
1. Get user-defined env vars from app_service.env (JSON blob)
2. Query service_dependencies WHERE app_service_id = ?
3. For each dependency record:
   a. Read dependency_service_type, dependency_column, env_var_name
   b. Query target table by dependency_service_id
   c. Extract column value → env_value
   d. Merge into env vars map: env[env_var_name] = env_value
4. Dependency values override user-defined values on key conflict
5. Pass final merged env map to deploy job
```

## Background Update Routine

Triggered when a service's connectable columns change (e.g., Postgres password rotated → `internal_url` re-generated):

1. Handler that updates the service spawns a goroutine
2. Goroutine queries `service_dependencies WHERE dependency_service_id = ?`
3. For each record, extracts new column value from changed service → updates `env_value`
4. `env_value` in dependency table is now current
5. Affected application services do NOT auto-redeploy — they pick up new value on next deploy

## Preview Instance Creation

```
1. Clone all services from production instance → preview instance
2. Track mapping: old_service_id → new_service_id
3. For each application service in preview:
   a. Query service_dependencies WHERE app_service_id = production_app_id
   b. Clone each record with:
      - new app_service_id = preview app service ID
      - new dependency_service_id = mapped preview dependency service ID
      - same dependency_service_type, dependency_column, env_var_name
      - new env_value = resolved from preview dependency service's current column value
4. Preview deploy-time resolution uses preview dependency records → preview services connect correctly
```

## Service Deletion Cleanup

When any service is deleted:
1. Delete `service_dependencies WHERE app_service_id = ?` (this service's outgoing connections)
2. Delete `service_dependencies WHERE dependency_service_id = ?` (other services' connections to this service)

## Conflict Resolution

Rule: **Dependency env vars override user-defined env vars.**

If user has manual env `DATABASE_URL=postgres://old` AND dependency connection for `DATABASE_URL` pointing to mydb `internal_url`:
- At deploy time: dependency value wins → `DATABASE_URL=postgres://user:pass@mydb-abc:5432/mydb`
- UI shows a warning indicator on conflicting vars

## Edge Cases

1. **Deleted dependency service**: When target service deleted → cleanup deletes all `service_dependencies` records referencing it. Dependent app's next deploy → no env var injected → app likely fails. Acceptable — user deleted the service.
2. **Circular dependencies**: Not prevented at schema level (A depends on B, B depends on A). Deploy-time resolution is direct only — no transitive chain. Each service resolves its own direct dependencies. Circular deps are a user configuration error that resolves without infinite loops.
3. **Stale cache before bg update**: If dependency changes but goroutine hasn't updated `env_value` yet → deploy job should resolve live from dependency table, not trust cached `env_value`. Cache is fallback, not source of truth.
4. **Preview creation without connections**: If `service_dependencies` is empty → no injection happens → preview created normally with manual env config only.
5. **Dependency service not ready**: If dependency service hasn't been deployed yet (no swarm service running) → `internal_url` still exists in DB → deploy-time resolution works. Actual runtime connectivity depends on both services being deployed.
6. **Duplication prevention**: `UNIQUE(app_service_id, env_var_name)` prevents two connections writing to same env var name from one service.
7. **Column not found**: If `dependency_column` value doesn't match any column in target service type → deploy-time resolution returns error → deploy fails with clear error message.

## Modules

- **Service Dependency Module**: Owns CRUD for dependency records, deploy-time env var merging, background value update routine, preview dependency cloning, service deletion cleanup.
- **Frontend**: Connection management UI (add/edit/delete), dependency graph visualization, env var conflict warnings, toast reminders.

## Implementation Decisions

- Dependency table has FK only to `app_service` (not to `psql_service` or `redis_service`) because: app services are the only dependents in V1. Predefined services don't connect outward.
- `dependency_service_id` stored as TEXT without FK because it can reference any of three service tables. Resolution uses `dependency_service_type` column.
- `env_value` is updated by background goroutine when dependency changes, not resolved fresh every deploy — tradeoff: fast deploy (no extra query in hot path) vs. potential staleness if goroutine slow. Actual deploy-time resolution should query live as safety measure.
- Dependency env vars override user env vars. Rationale: the dependency connection represents user's explicit "manage this for me" intent — should not be accidentally shadowed by a stale manual env var.
- Background update routine uses goroutines, not a job queue. Lightweight enough for single-instance Godploy. If scaling needed later, move to worker-based job processing.
- Preview dependency cloning happens after all services are cloned but before preview deploy — dependency table acts as the wiring layer between cloned services.

## Testing Decisions

- Test deploy-time env var merging: verify dependency values injected, override behavior on conflict, edge case with empty dependency table
- Test background update: verify `env_value` changes when dependency service's connectable columns change
- Test preview cloning: verify dependency records created with correct rewritten IDs and resolved values
- Test service deletion cleanup: verify dependency records removed for both outgoing and incoming references
- Test handler-level: API returns 200/400 correctly, UNIQUE constraint on duplicates, toast message content
- Prefer handler-level integration tests (existing pattern in `backend/integration_tests/`)
- Module-level unit tests for resolution logic, cloning logic, cleanup logic

## Out of Scope

- Transitive dependency resolution (A → B → C chains)
- Auto-redeploy of dependent services when dependency changes
- Dependency-based selective preview (deploy only services affected by a PR, based on dependency graph)
- Non-app services as dependents (psql, redis don't declare dependencies outward)
- Runtime health tracking of dependency connections
- Import/export of dependency graph
- CLi support for managing connections
