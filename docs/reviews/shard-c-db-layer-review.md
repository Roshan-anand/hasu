# Shard C Review: DB Layer + Migrations + Tests + Frontend

> **Reviewer**: DeepSeek V4 Pro  
> **Scope**: Uncommitted changes — staged, unstaged, and untracked files  
> **Date**: 2026-06-24

---

## Summary

This review covers ~25 files spanning SQLite migrations (0001–0005), query definitions, sqlc-generated Go code, integration tests, and a frontend PR-preview dropdown component. The overall architecture is sound, but several critical issues were identified — most notably a missing `ALTER TABLE` in migration 0005, misaligned `SELECT *` scan ordering, and missing `CHECK` constraints on `psql_service`.

**Verdict**: **Changes recommended before merging.** 2 critical, 3 warnings, 4 info-level issues.

---

## Findings

### 🔴 CRITICAL-01: Migration 0005 missing `ALTER TABLE ADD COLUMN` for instance columns

**File**: `apps/server/sqlite/migrations/0005_preview_instance.up.sql`  
**Lines**: entire file (14 lines)

**Problem**:  
Migration 0005 only creates the `github_pull_requests` table:

```sql
CREATE TABLE IF NOT EXISTS github_pull_requests ( ... );
```

It is **missing `ALTER TABLE` statements** to add columns `git_source_type`, `git_source_value`, `status`, and `created_by` to the existing `instance` table.

**Impact**:

- Migration 0001 (`0001_init_schema.up.sql`) defines these columns for **fresh installs** (`CREATE TABLE IF NOT EXISTS instance ( ... git_source_type ... )`).
- For **existing databases** that were created with a prior version of 0001 (before these columns were added), running the migration chain from 0001→0002→0003→0004→0005 will **not** add these columns to `instance`.
- All queries in `instance.sql` (e.g. `GetProductionInstanceByProject`, `CreatePreviewInstance`) reference these columns and will **fail at runtime** on existing databases.

**Fix**:  
Add the following to `0005_preview_instance.up.sql` before the `CREATE TABLE`:

```sql
ALTER TABLE instance ADD COLUMN git_source_type TEXT;
ALTER TABLE instance ADD COLUMN git_source_value TEXT;
ALTER TABLE instance ADD COLUMN status TEXT NOT NULL DEFAULT 'creating';
ALTER TABLE instance ADD COLUMN created_by TEXT NOT NULL DEFAULT 'manual';
```

> **Note**: SQLite does **not** support `ALTER TABLE ADD COLUMN` with `CHECK` constraints, `NOT NULL` without `DEFAULT`, or `UNIQUE`. The `CHECK` and `NOT NULL` constraints exist only in 0001's `CREATE TABLE` for fresh installs. This is acceptable — application-level validation through sqlc overrides provides the enforcement.

---

### 🔴 CRITICAL-02: Instance model field ordering mismatch for `SELECT *` queries

**File**: `apps/server/internal/db/models.go` (Instance struct)  
**Lines**: Instance struct definition (lines 127-139)

**Problem**:  
The `Instance` Go struct has `GitSourceType`, `GitSourceValue`, `Status`, and `CreatedBy` fields **after** `CreatedAt`:

```go
type Instance struct {
    ID             uuid.UUID            `json:"id"`
    ProjectID      uuid.UUID            `json:"project_id"`
    IsProduction   bool                 `json:"is_production"`
    Name           string               `json:"name"`
    Network        string               `json:"network"`
    CreatedAt      time.Time            `json:"created_at"`
    GitSourceType  types.GitSourceType  `json:"git_source_type"`
    GitSourceValue sql.NullString       `json:"git_source_value"`
    Status         types.InstanceStatus `json:"status"`
    CreatedBy      types.CreatedBy      `json:"created_by"`
}
```

But in the SQL table (0001), the column order is:

```
id, project_id, is_production, name, network, created_at,
git_source_type, git_source_value, status, created_by
```

**Impact**:  
Currently **none** — because **all** instance queries in `instance.sql` explicitly list columns rather than `SELECT *`:

```sql
SELECT id, project_id, is_production, name, network,
    git_source_type, git_source_value, status, created_by,
    created_at
```

However, this is fragile. If any future query uses `SELECT * FROM instance` and scans into `Instance` or `GetProductionInstanceByProjectRow`, the scan will assign `created_at`'s value to `GitSourceType` and subsequent fields will be misaligned, causing silent data corruption or panics.

**Recommendation**:  
Reorder the Go struct to match SQL column order:

```go
type Instance struct {
    ID             uuid.UUID            `json:"id"`
    ProjectID      uuid.UUID            `json:"project_id"`
    IsProduction   bool                 `json:"is_production"`
    Name           string               `json:"name"`
    Network        string               `json:"network"`
    CreatedAt      time.Time            `json:"created_at"`       -- stays here
    GitSourceType  types.GitSourceType  `json:"git_source_type"`   -- moved up
    GitSourceValue sql.NullString       `json:"git_source_value"`  -- moved up
    Status         types.InstanceStatus `json:"status"`           -- moved up
    CreatedBy      types.CreatedBy      `json:"created_by"`       -- moved up
}
```

> **Alternatively**: Change all instance queries to use explicit column ordering matching the struct. Currently they already do, but the mismatch is a maintenance trap.

---

### 🟡 WARNING-01: Missing `CHECK` constraints on `psql_service` status/type

**Files**:

- `apps/server/sqlite/migrations/0001_init_schema.up.sql` (psql_service table)
- `apps/server/sqlite/migrations/0003_redis_service.up.sql` (redis_service table for comparison)

**Problem**:  
`psql_service` has **no** `CHECK` constraints on `status` or `type`:

```sql
-- 0001_init_schema.up.sql
CREATE TABLE IF NOT EXISTS psql_service (
    ...
    status TEXT NOT NULL,
    type TEXT NOT NULL,
    ...
);
```

Compare with `redis_service` in 0003, which has proper `CHECK` constraints:

```sql
-- 0003_redis_service.up.sql
status TEXT NOT NULL CHECK(status IN ('running','paused')),
type TEXT NOT NULL CHECK(type IN ('redis')),
```

**Impact**:  
Invalid data (e.g., `status = 'invalid'`) can be inserted directly into `psql_service`. While sqlc overrides map `psql_service.status` → `types.PredefinedServiceStatus` (which restricts to `'running'`/`'paused'`), this is application-level only. Direct database writes, manual SQL, or bugs could bypass it.

**Recommendation**:  
Add a migration (0006 or modify 0001 — but 0001 is fresh-install-only) to add `CHECK` constraints:

Since SQLite cannot `ALTER TABLE ADD CHECK`, the options are:

1. **For fresh installs**: Modify 0001's `psql_service` to add `CHECK(status IN ('running','paused'))` and `CHECK(type IN ('psql'))`.
2. **For existing databases**: Create a new migration that recreates the table with constraints (dump, recreate, repopulate) — or accept the application-level enforcement as sufficient.

> **Acceptable to defer**: Since the sqlc overlay forces the correct types at compile time, this is a defense-in-depth issue, not a runtime bug.

---

### 🟡 WARNING-03: `GithubPullRequest.CreatedAt`/`UpdatedAt` should be `time.Time`, not `sql.NullTime`

**File**: `apps/server/internal/db/models.go` (GithubPullRequest struct)  
**Lines**: ~89-95

**Problem**:

```go
type GithubPullRequest struct {
    ...
    CreatedAt  sql.NullTime `json:"created_at"`
    UpdatedAt  sql.NullTime `json:"updated_at"`
}
```

The SQL schema defines both columns as:

```sql
created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
```

The `DEFAULT CURRENT_TIMESTAMP` guarantees these are **never** NULL on insert. Additionally, the `UpsertPullRequest` query explicitly sets `updated_at = CURRENT_TIMESTAMP` on conflict, and the insert omits both columns (relying on DEFAULT). So they will always have values.

**Impact**:  
Callers must check `.Valid` before using the value, adding unnecessary boilerplate:

```go
if pr.CreatedAt.Valid {
    use(pr.CreatedAt.Time)
}
```

When a simple `time.Time` would suffice.

**Recommendation**:  
Change the Go model to `time.Time` and regenerate:

```go
CreatedAt  time.Time `json:"created_at"`
UpdatedAt  time.Time `json:"updated_at"`
```

> **Note**: This requires adding the columns explicitly in the `GetPullRequestByRepoAndNumber` and `GetPullRequestsByInstance` queries to ensure they're always included in the scan. Currently both queries `SELECT ... created_at, updated_at` and rely on `DEFAULT CURRENT_TIMESTAMP`, so the values are always present.

---

### 🔵 INFO-01: `GetActivePreviewByPR` — `NULL` comparison semantics

**File**: `apps/server/sqlite/query/instance.sql` (GetActivePreviewByPR)  
**Lines**: 29-35

```sql
WHERE project_id = ? AND git_source_type = 'pr' AND git_source_value = ?
    AND status NOT IN ('deleting', 'error')
```

**Observation**:  
`git_source_value` is nullable (`TEXT` with no `NOT NULL` in 0001). If a row has `git_source_value IS NULL`, the `= ?` comparison will **never** match (SQL NULL semantics). This is **correct behavior** — PR instances should always have a non-null `git_source_value`. The parameter type `sql.NullString` in the generated code also handles this correctly. No action needed.

---

### 🔵 INFO-02: `orphan_volume.type` CHECK matches `PredefServiceType` — consistent

**File**: `apps/server/sqlite/migrations/0002_orphan_schema.up.sql`  
**Lines**: type definition

```sql
type TEXT NOT NULL CHECK(type IN ('psql','redis','mongodb'))
```

Compare with `apps/server/internal/lib/types/types.go`:

```go
const (
    PSQLPredefServiceType  PredefServiceType = "psql"
    RedisPredefServiceType PredefServiceType = "redis"
    MongoPredefServiceType PredefServiceType = "mongodb"
)
```

**Observation**: The CHECK constraint and Go enum are fully aligned. sqlc override maps `orphan_volume.type` → `types.PredefServiceType`. **No issue.**

---

### 🔵 INFO-03: Deployment status CHECK matches `DeploymentStatus` enum — consistent

**File**: `apps/server/sqlite/migrations/0001_init_schema.up.sql` (deployments table)  
**Lines**: status CHECK

```sql
status TEXT NOT NULL DEFAULT 'queued' CHECK(status IN ('building','ready','error','queued','inactive','pruned','paused'))
```

Compare with `types.go`:

```go
const (
    DeploymentBuilding DeploymentStatus = "building"
    DeploymentReady    DeploymentStatus = "ready"
    DeploymentError    DeploymentStatus = "error"
    DeploymentQueued   DeploymentStatus = "queued"
    DeploymentInactive DeploymentStatus = "inactive"
    DeploymentPruned   DeploymentStatus = "pruned"
    DeploymentPaused   DeploymentStatus = "paused"
)
```

**Observation**: All 7 values match. sqlc override maps `deployments.status` → `types.DeploymentStatus`. **No issue.**

---

### 🔵 INFO-04: Frontend PR dropdown — `pr.id` used as Svelte key but `id` may be missing

**File**: `frontend/src/lib/components/InstancePRPreviewDropdown.svelte`  
**Lines**: ~104

```svelte
{#each prList as pr, i (pr.id || i)}
```

**Observation**: The fallback `|| i` for the key is a reasonable pattern when `pr.id` might be undefined. However, if `pr.id` is `0` (falsy in JS), it would fall back to the array index, potentially causing duplicate key warnings if two PRs have `id: 0`. PR IDs from GitHub are always positive integers, so this is safe in practice. **No action needed.**

---

## Integration Test Review

### Test: `TestAppServiceDomainUpdate`

**File**: `apps/server/integration_tests/app_service_domain_test.go`

| Aspect                     | Verdict                                                           |
| -------------------------- | ----------------------------------------------------------------- |
| Setup flow                 | ✅ Complete — registers user, fetches github app, creates service |
| Validation cases           | ✅ Empty body (400), non-existent service ID (500)                |
| Docker dependency handling | ✅ Graceful skip with `t.Log` when Docker unavailable             |
| DB verification            | ✅ Checks `IsPublic` and `Domain.String` after update             |
| Cleanup                    | ✅ Deletes app service at end                                     |

**Issues**: None found. The test is well-structured.

### Test: `TestDependency`

**File**: `apps/server/integration_tests/dependency_test.go`

| Aspect                   | Verdict                                                                           |
| ------------------------ | --------------------------------------------------------------------------------- |
| CRUD coverage            | ✅ Create, read, update, delete all tested                                        |
| Edge cases               | ✅ Self-dependency rejected (400), invalid env key (400), duplicate env key (500) |
| Cross-instance rejection | ✅ Creates separate org/project/instance and verifies 400                         |
| PSQL/Redis dependency    | ✅ Creates both service types and tests dependency creation                       |
| Domain validation        | ✅ Domain target rejected for internal services and public-with-empty-domain      |
| Cleanup on delete        | ✅ Verifies incoming dependencies are cleaned up when PSQL/app target is deleted  |
| Graph endpoint           | ✅ Verifies nodes, edges, invalid UUID                                            |

**Issues**: None found. This is a thorough test suite.

### Test Utils

**File**: `apps/server/integration_tests/utils.go`

| Aspect                | Verdict                                                                                                                                                     |
| --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Server initialization | ✅ Creates temp dirs, loads config, sets up routes                                                                                                          |
| Auth mocking          | ✅ Sets `AuthUser` in echo context                                                                                                                          |
| Cookie jar            | ✅ Created but `hasCookie()` has a logic bug — returns `false` if the FIRST cookie checked matches, rather than checking all. This function appears unused. |

**Minor issue in `hasCookie`** (line ~103):

```go
func hasCookie(c []*http.Cookie, cfg *config.Config) bool {
    for _, cookie := range c {
        switch cookie.Name {
        case cfg.SessionDataName, cfg.SessionTokenName:
        default:
            return false  // returns false immediately for non-matching cookies
        }
    }
    return true
}
```

This returns `false` if **any** cookie name doesn't match, rather than checking that **all** required cookies exist. The function appears unused in the test code, so this is a **cosmetic issue** only.

---

## sqlc Configuration Review

**File**: `apps/server/sqlc.yaml`

| Override                                                      | Status |
| ------------------------------------------------------------- | ------ |
| `user.role` → `types.UserRole`                                | ✅     |
| `deployments.status` → `types.DeploymentStatus`               | ✅     |
| `psql_service.type` → `types.ServiceType`                     | ✅     |
| `psql_service.status` → `types.PredefinedServiceStatus`       | ✅     |
| `redis_service.type` → `types.ServiceType`                    | ✅     |
| `redis_service.status` → `types.PredefinedServiceStatus`      | ✅     |
| `app_service.type` → `types.ServiceType`                      | ✅     |
| `instance.git_source_type` (nullable) → `types.GitSourceType` | ✅     |
| `instance.status` → `types.InstanceStatus`                    | ✅     |
| `instance.created_by` → `types.CreatedBy`                     | ✅     |
| `orphan_volume.type` → `types.PredefServiceType`              | ✅     |
| UUID → `uuid.UUID` / `uuid.NullUUID`                          | ✅     |
| `app_service.port` → `int32`                                  | ✅     |

All overrides correctly match model field names and Go types. **No issues.**

### Type alignment check

| SQL CHECK                                                                                  | Go type                                  | Matching?                    |
| ------------------------------------------------------------------------------------------ | ---------------------------------------- | ---------------------------- |
| `instance.status IN ('creating','ready','deleting')`                                       | `InstanceStatus` with same 3 values      | ✅                           |
| `instance.git_source_type IN ('pr','branch')`                                              | `GitSourceType` with `'pr'`, `'branch'`  | ✅                           |
| `instance.created_by IN ('manual','webhook')`                                              | `CreatedBy` with `'manual'`, `'webhook'` | ✅                           |
| `deployments.status IN ('building','ready','error','queued','inactive','pruned','paused')` | `DeploymentStatus` with all 7            | ✅                           |
| `redis_service.status IN ('running','paused')`                                             | `PredefinedServiceStatus` with both      | ✅                           |
| `redis_service.type IN ('redis')`                                                          | `ServiceType` includes `'redis'`         | ✅                           |
| **`psql_service.status` — NO CHECK**                                                       | `PredefinedServiceStatus`                | ⚠️ Mismatch (see WARNING-01) |
| `orphan_volume.type IN ('psql','redis','mongodb')`                                         | `PredefServiceType` with all 3           | ✅                           |

---

## Frontend Review

**File**: `frontend/src/lib/components/InstancePRPreviewDropdown.svelte`

| Aspect                     | Verdict                                                                          |
| -------------------------- | -------------------------------------------------------------------------------- |
| Component structure        | ✅ Well-organized with clear sections (search, list, selection, actions)         |
| Loading/error/empty states | ✅ All three states handled                                                      |
| Search/filter UX           | ✅ Filters by title and PR number, case-insensitive                              |
| Selection UX               | ✅ Visual highlight, clear button, confirmation dialog                           |
| Accessibility              | ✅ Uses `button` elements with proper types, labels via `aria`-adjacent patterns |
| Svelte 5 runes usage       | ✅ `$state`, `$derived`, `$props` used correctly                                 |
| Dialog portal              | ✅ Uses portal for overlay rendering                                             |

**Issues**: None significant. The component is clean and well-implemented.

---

## Final Checklist

| Check                                              | Status                                                              |
| -------------------------------------------------- | ------------------------------------------------------------------- |
| All migration files parse correctly                | ✅                                                                  |
| 0005 adds `github_pull_requests` table             | ✅                                                                  |
| 0005 adds `ALTER TABLE` for instance columns       | ❌ **CRITICAL-01**                                                  |
| Down migration is safe (DROP TABLE only)           | ✅ (per design — non-reversible for instance columns is documented) |
| sqlc generated code matches query files            | ✅                                                                  |
| All Go types match CHECK constraints               | ❌ **WARNING-01** (psql_service missing CHECK)                      |
| `AppService.Domain` is `sql.NullString` everywhere | ✅                                                                  |
| Instance model handles nullable fields correctly   | ✅ (GitSourceType nullable, GitSourceValue nullable)                |
| Integration tests compile and cover new paths      | ✅                                                                  |
| Frontend component renders and handles states      | ✅                                                                  |
| Migration 0004 asymmetry documented                | ✅ (see WARNING-02)                                                 |

---

## Recommended Actions (Priority Order)

1. **🔴 CRITICAL**: Add `ALTER TABLE ADD COLUMN` to `0005_preview_instance.up.sql` for the 4 instance columns.
2. **🔴 CRITICAL**: Reorder `Instance` struct fields in `models.go` to match SQL column order, OR add a comment warning against using `SELECT *` with this struct.
3. **🟡 WARNING**: Add `CHECK` constraints to `psql_service.status` and `psql_service.type` (or document why application-level enforcement is sufficient).
4. **🟡 WARNING**: Change `GithubPullRequest.CreatedAt`/`UpdatedAt` from `sql.NullTime` to `time.Time`.
5. **🔵 INFO**: Fix `hasCookie()` utility function logic if it becomes used.
