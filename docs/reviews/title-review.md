# Code Review: Preview Orchestration + Jobs Core (Shard A)

**Reviewer:** DeepSeek V4 Pro  
**Date:** 2026-06-25  
**Scope:** `preview.go`, `preview_utils.go`, `app_worker.go`, `app_worker_utils.go`, `predef_worker.go`, `core.go` (submit[T] dispatcher)

---

## Executive Summary

The preview instances implementation is structurally sound and follows the existing worker-assigner pattern well. `submit[T]` is a clean generic dispatcher. The sync `DeployPredefinedService` extraction is correct. However, **two critical bugs** in cleanup and webhook flows will cause data/resource leaks in production, and there are several important design gaps vs. the PRD.

---

## Critical Bugs

### 🔴 C1: `GetAllService` returns empty `swarm_service` for psql/redis → cleanup silently skips predefined services

**File:** `apps/server/sqlite/query/service.sql` (lines 9-10, 19-20)  
**Cascading to:** `apps/server/internal/jobs/deployment/preview_utils.go` lines 33-40

**Problem:**  
The `GetAllService` query uses `'' AS swarm_service` for the psql and redis UNION arms instead of the actual `ps.swarm_service` / `rs.swarm_service` column:

```sql
-- psql arm:
SELECT
    ps.id, ps.type, ps.name, ps.status, ps.volume,
    '' AS gh_repo_name, '' AS gh_repo_url,
    '' AS git_provider, '' AS branch_name,
    '' AS swarm_service,   -- ← BUG: should be ps.swarm_service
    ps.created_at
FROM psql_service ps
WHERE ps.instance_id = @instance_id
```

`collectCleanupResources` in `preview_utils.go` checks `svc.SwarmService != ""` — but for psql/redis services this will **always be empty**. Result: **Docker swarm services for PSQL and Redis are never removed during preview cleanup**, causing resource leaks.

**Fix:** Replace `'' AS swarm_service` with `ps.swarm_service` and `rs.swarm_service` in both psql/redis UNION arms. Also verify the `GetAllServiceRow` struct fields order matches the new SELECT columns.

---

### 🔴 C2: Webhook creates preview with `uuid.Nil` for `ProjectID` → `GetProductionInstanceByProject` fails or returns wrong instance

**File:** `apps/server/internal/handlers/github.go` lines 506-513

**Problem:**  
The webhook `pull_request.opened`/`reopened` handler passes `ProjectID: uuid.Nil`:

```go
_ = h.Server.Services.Deployment.AssignCreatePreview(h.qCtx, &deployjob.CreatePreviewJobParams{
    ProjectID: uuid.Nil,  // ← CRITICAL: never resolved
    Name:      fmt.Sprintf("pr-%d", pr.GetNumber()),
    ...
})
```

`CreatePreviewFromPR` uses `q.GetProductionInstanceByProject(ctx, input.ProjectID)` which will either return `sql.ErrNoRows` or a random UUID-zero match (UUID zero is `00000000-0000-0000-0000-000000000000`). The `issue_comment` handler has the same issue (line 597).

**Fix:** Resolve `ProjectID` from the repo/webhook context before queuing:

```go
projectID, err := q.GetProjectIDByRepoID(ctx, repo.GetID())
if err != nil { /* log and return */ }
```

---

### 🔴 C3: `HeadBranch` required validation fails for webhook-triggered previews

**File:** `apps/server/internal/handlers/github.go` line 597  
**File:** `apps/server/internal/jobs/deployment/core.go` lines 70-71

**Problem:**  
`CreatePreviewJobParams.HeadBranch` has `validate:"required"`. The `issue_comment` handler passes `HeadBranch: ""`. The `submit[T]` function calls `d.v.Struct(body)` which will **reject the job** silently.

**Fix:** Either make `HeadBranch` not required (with conditional validation), or resolve the head branch from the PR before submitting.

---

## High Severity Issues

### 🟠 H1: No DB transaction wrapping the clone operations

**File:** `apps/server/internal/jobs/deployment/preview.go` lines 72-97

**Problem:**  
`clonePsqlServices`, `cloneRedisServices`, `cloneAppServices`, and `cloneDependencies` each execute individual INSERT statements. If any step fails mid-way (e.g., `cloneRedisServices` fails after `clonePsqlServices` succeeds), partial state remains in the DB — orphaned app services, psql services, and Docker resources with no cleanup.

The PRD explicitly says **"B3 — Clone Services (DB transaction)"**.

**Fix:** Wrap the entire clone block (after instance record creation, before `triggerAppServiceDeploys`) in a `sql.Tx`. On failure, roll back the transaction and remove any Docker resources already created.

---

### 🟠 H2: `cleanupPreview` recovery and error handling gap

**File:** `apps/server/internal/jobs/deployment/preview_utils.go` lines 34-62

**Problems:**
1. **No panic recovery** — if any panic occurs inside the goroutine, the entire process crashes.
2. **Silent partial cleanup** — if `GetAllService` fails (line 37), it only removes the network and deletes the DB record but skips removing Docker services/volumes, leaving orphaned resources.
3. **Sequential volume removal ignores errors** — `RemoveVolume` errors are swallowed with `continue`.

**Fix:**
```go
func (d *DeploymentService) cleanupPreview(ctx context.Context, previewID uuid.UUID, network string) {
    defer func() {
        if r := recover(); r != nil { log.Printf("panic in cleanupPreview: %v", r) }
    }()
    // ... proceed with best-effort cleanup even on partial errors
}
```

---

### 🟠 H3: `status='ready'` set immediately after queueing deploys, not after completion

**File:** `apps/server/internal/jobs/deployment/preview.go` line 99

**Problem:**  
```go
if err := q.UpdateInstanceStatus(ctx, db.UpdateInstanceStatusParams{ID: previewID, Status: types.InstanceReady}); err != nil {
```

This is called right after `triggerAppServiceDeploys` submits jobs to channels. If a deployment pipeline subsequently fails (build error, Docker failure), the instance shows `ready` with broken services. The PRD §5.1 step 6 says: *"Instance `status` transitions to `ready` when all queued deploy jobs finish."*

**Fix:** Either:
- Track completion of all queued jobs and update status afterward, or
- Keep status as `creating` and transition to `ready` from within each successful deploy, with a final check that all services are healthy.

---

### 🟠 H4: `runCloneDeployPipeline` creates the network that already exists

**File:** `apps/server/internal/jobs/deployment/app_worker.go` lines 177-184

**Problem:**  
`CreateNetwork` is called inside `runCloneDeployPipeline`. The network was already created in `CreatePreviewFromPR` (preview.go line 58). While `CreateNetwork` is idempotent (checks existence), this is a logic error — the clone-deploy pipeline should not be managing network lifecycle. If the network was somehow deleted between creation and deployment, recreating it here creates a race condition.

**Fix:** Remove `CreateNetwork` from `runCloneDeployPipeline` — the network is guaranteed to exist by the caller.

---

## Medium Severity Issues

### 🟡 M1: `InstanceStatus` missing `"error"` state

**File:** `apps/server/internal/lib/types/types.go` lines 34-40

**Problem:**  
The PRD schema defines `CHECK(status IN ('creating','ready','updating','deleting','error'))`. The code only defines `InstanceCreating`, `InstanceReady`, `InstanceDeleting`. There's no `InstanceError` type. The `GetActivePreviewByPR` query filters out `'error'` but the constant is never used.

**Fix:** Add `InstanceError InstanceStatus = "error"`.

---

### 🟡 M2: `deployData.port` hardcoded to 80 in preview flows

**File:** `apps/server/internal/jobs/deployment/app_worker_utils.go` lines 140-149

**Problem:**  
```go
func (d *DeploymentServiceParams) getDeployData(network string) *deployData {
    return &deployData{
        // ...
        port: 80,  // ← ignores service's actual port
    }
}
```

And in `runCloneDeployPipeline` (line 196), the port from `CloneDeployData` is used correctly via `data.Port`. But the full deploy pipeline (`runDeploymentPipeline` → `getDeployData`) hardcodes port 80, meaning **PR-matched services in previews always deploy on port 80** regardless of the production service's port setting.

**Fix:** Add `port` to `DeploymentServiceParams` and thread it through to `getDeployData`.

---

### 🟡 M3: Thread safety — `DockerClient.ServiceCreate` and `ServiceRemove` called concurrently without coordination

**File:** `apps/server/internal/jobs/deployment/preview.go` (clone methods)  
**File:** `apps/server/internal/jobs/deployment/preview_utils.go` (cleanup runs in goroutine)

**Problem:**  
Preview creation and deletion can overlap. `CreatePreviewFromPR` is called from a worker sequentially, but `DeletePreview` launches `cleanupPreview` in a goroutine. If a user deletes a preview concurrently while creation is still deploying services, Docker swarm operations race.

**Fix:** Add a mutex or state machine on the preview instance (check status before cleanup).

---

### 🟡 M4: `cloneDependencies` silently skips missing ID mappings

**File:** `apps/server/internal/jobs/deployment/preview.go` lines 220-225

**Problem:**  
```go
newSource, ok1 := idMap[dep.SourceServiceID]
newTarget, ok2 := idMap[dep.TargetServiceID]
if !ok1 || !ok2 {
    continue  // ← silently drops dependency
}
```

If a dependency references a service that wasn't cloned (shouldn't happen, but possible due to bugs), the dependency is silently dropped instead of being logged/errored.

**Fix:** Log a warning when skipping and consider returning an error (fail-fast is safer for data integrity).

---

### 🟡 M5: Orphaned Docker network if instance DB record creation fails

**File:** `apps/server/internal/jobs/deployment/preview.go` lines 58-71

**Problem:**  
Current order is correct per invariant 9 (network first) — but if `CreatePreviewInstance` (the INSERT) fails after `CreateNetwork` succeeds, the Docker network is orphaned with no cleanup path.

**Fix:** Add a deferred cleanup in case of error:

```go
networkCreated := false
if err := d.docker.CreateNetwork(previewNetwork); err != nil { ... }
networkCreated = true
// ... later, on any error:
if networkCreated {
    d.docker.RemoveNetwork([]string{previewNetwork})
}
```

---

## Low Severity / Minor Issues

### 🔵 L1: `GetAllService` doesn't return `swarm_service` for psql/redis in the query (related to C1)

Already covered in C1 above.

### 🔵 L2: `GetActivePreviewByPR` filters by `NOT IN ('deleting', 'error')` but `'error'` never set

Status `'error'` is never written to any instance. The SQL filter is harmless but dead code. If `InstanceError` is added per M1, the filter becomes meaningful.

### 🔵 L3: `slugify` is simplistic — doesn't handle special characters or unicode

**File:** `apps/server/internal/jobs/deployment/preview.go` lines 252-256

```go
func slugify(name string) string {
    name = strings.ToLower(strings.TrimSpace(name))
    name = strings.ReplaceAll(name, " ", "-")
    return name
}
```

No handling of: multiple consecutive hyphens, leading/trailing hyphens, non-ASCII characters, uppercase in network names (Docker overlay networks are case-sensitive). Consider using `url.PathEscape` or a proper slug library.

### 🔵 L4: `DockerClient.RemoveVolume` doesn't check if name is empty

**File:** `apps/server/internal/lib/docker/docker.go` lines 110-115

If `volumeName` is `""`, Docker's `VolumeRemove` will fail with an error. The cleanup code checks `svc.Volume != ""` before collecting, so this is safe currently — but defensive validation would prevent future bugs.

### 🔵 L5: PRD §12 mentions `apps/server/internal/service/preview.go` — actual code lives in `jobs/deployment/preview.go`

The PRD says the orchestration logic should be in `apps/server/internal/service/preview.go`. Instead, it's implemented as methods on `DeploymentService` in `jobs/deployment/`. This is not a bug per se, but a design deviation worth documenting. The orchestration mixes job-queue concerns with business logic. If preview logic needs to be tested independently, it should be extractable without the `DeploymentService` worker infrastructure.

### 🔵 L6: `RebuildPreviewOnPush` is a no-op stub

**File:** `apps/server/internal/jobs/deployment/preview_utils.go` lines 65-68

```go
func (d *DeploymentService) RebuildPreviewOnPush(ctx context.Context, previewID uuid.UUID, repoID int, branch string) error {
    // todo: find PR-matched service and trigger rebuild
    return nil
}
```

The webhook `synchronize` handler calls this (github.go line 523) and the `issue_comment` handler also calls it (line 593). Since it's a no-op, PR push updates and comment-triggered rebuilds are silently ignored. This should either be implemented or the callers should log a warning.

---

## PRD Compliance Check

| PRD Requirement | Status | Notes |
|----------------|--------|-------|
| B1: Validate production exists | ✅ | `GetProductionInstanceByProject` errors handled |
| B2: Create preview instance record | ✅ | Done before clone operations |
| B3: Clone services in DB transaction | ❌ **H1** | No wrapping transaction |
| B4: Clone dependencies with ID remapping | ✅ | `cloneDependencies` with idMap |
| B5: Commit & Queue deploys after deps | ✅ | `triggerAppServiceDeploys` called after `cloneDependencies` |
| Phase D: Set status=deleting → cleanup | ⚠️ Partial | C1 bug breaks predefined service cleanup |
| Predef deploy worker extracted as sync fn | ✅ | `DeployPredefinedService` correctly extracted |
| `runCloneDeployPipeline` skips build | ✅ | Uses existing image, calls `ServiceCreate` |
| Async/queued operations after instance creation | ✅ | Jobs submitted to channels |
| Domain: `<preview-slug>.<original-domain>` | ✅ | Implemented in `cloneAppServices` |
| No duplicate active preview | ❌ **Missing check** | Handler doesn't check before submitting |
| PRD §12: `service/preview.go` path | ⚠️ Mismatch | Code in `jobs/deployment/preview.go` |
| `status='error'` in schema | ❌ **M1** | Missing from types |

---

## Summary

**Critical (fix before merge):** C1 (swarm_service empty for psql/redis → cleanup leak), C2 (uuid.Nil project_id in webhook), C3 (empty HeadBranch fails validation)

**High (fix before merge):** H1 (no clone transaction), H2 (cleanup panic recovery), H3 (ready set before deploys complete), H4 (network recreation in clone pipeline)

**Medium (fix this sprint):** M1-M5

**Low (document / backlog):** L1-L6
