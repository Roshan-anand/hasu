# Review: Shard B — Handlers + Routes + Types

## Overview

Review of uncommitted changes across handlers, routes, types, and sqlc overrides for the preview instance feature. 14 files examined.

---

## 🔴 Critical Issues

### C1. `HeadBranch=""` silently fails validation in issue_comment webhook

**Files:** `github.go` (line ~680), `core.go` (line 57, 278)

**Problem:**  
The `issue_comment` handler passes `HeadBranch: ""` to `AssignCreatePreview` because the comment event does not contain the PR head branch. However, `CreatePreviewJobParams.HeadBranch` is tagged `validate:"required"`. The `submit[T]` generic dispatcher calls `d.v.Struct(body)` which **rejects** `HeadBranch=""` with a validation error. The webhook handler ignores the error with `_ = ...`, so the preview creation **silently fails** — no log, no error response, no observable effect.

```go
// core.go line 57 — the struct tag
type CreatePreviewJobParams struct {
    HeadBranch string `validate:"required"`  // ← blocks ""
    ...
}

// github.go line ~680 — the caller
_ = h.Server.Services.Deployment.AssignCreatePreview(h.qCtx, &deployjob.CreatePreviewJobParams{
    ...
    HeadBranch:     "",  // ← validation rejects this
    ...
})
```

Additionally, even if `submit[T]` were bypassed, `CreatePreviewFromPR` (line 14 of `preview.go`) re-validates the same struct, hitting the same failure.

**Fix:**  
Either (a) remove `validate:"required"` from `HeadBranch` and handle empty in the orchestrator, or (b) resolve the head branch from the PR cache before calling `AssignCreatePreview`.

---

### C2. `RebuildPreviewOnPush` is a no-op stub

**Files:** `preview_utils.go` (line ~82), `github.go` (lines ~610, ~695)

**Problem:**  
`DeploymentService.RebuildPreviewOnPush` is implemented as:

```go
func (d *DeploymentService) RebuildPreviewOnPush(ctx context.Context, previewID uuid.UUID, repoID int, branch string) error {
    // todo: find PR-matched service and trigger rebuild
    return nil
}
```

This means:

- `pull_request.synchronize` → silently does nothing (line 610 of github.go)
- `issue_comment "/godploy deploy"` when preview exists → silently does nothing (line 695 of github.go)

Both paths call `RebuildPreviewOnPush` and discard its return value. Users pushing new commits or issuing `/godploy deploy` on an existing preview will observe zero effect.

**Fix:**  
Implement the todo — find the PR-matched app-service in the preview and queue `AssignRebuild`.

---

## 🟠 High Severity

### H1. `ListPreviews` omits authorization check

**File:** `preview.go` (lines 97–115)

**Problem:**  
`ListPreviews` accepts any `project_id` query parameter and returns preview data without verifying the caller owns that project. The route is behind auth middleware (`protected` group), so the caller is authenticated, but **any authenticated user can list previews for any project**.

```go
func (h *PreviewHandler) ListPreviews(c *echo.Context) error {
    projectID, err := uuid.Parse(c.QueryParam("project_id"))
    // ... no ownership check ...
    previews, err := h.Server.Services.Deployment.ListPreviews(h.qCtx, projectID)
    return c.JSON(http.StatusOK, types.Res[[]db.GetPreviewInstancesByProjectRow]{...})
}
```

This violates **Invariant #7** which explicitly states "Must verify caller owns the project."

**Fix:**  
Add a query (e.g., `CheckUserProjectAccess`) or join the project → org → user chain before returning data. The pattern already exists elsewhere (e.g., `org.go` uses `CheckUserOrgExists`).

---

### H3. Webhook `opened`/`reopened` not idempotent

**File:** `github.go` (lines ~588–601)

**Problem:**  
The `opened`/`reopened` case calls `AssignCreatePreview` unconditionally — it does **not** check whether a preview already exists for this repo+PR. GitHub webhooks are delivered at-least-once. A duplicate delivery would queue a second preview creation job.

```go
case "opened", "reopened":
    _ = h.Server.Services.Deployment.AssignCreatePreview(h.qCtx, ...)  // no existence check
```

The `synchronize` and `closed` cases both check via `GetActivePreviewByPR` first. `opened`/`reopened` should follow the same pattern.

**Fix:**  
Check `GetActivePreviewByPR` before queuing. If a preview already exists, treat as a no-op (idempotent).

---

## 🟡 Medium Severity

### M1. `InstanceStatus` missing `updating` and `error` values

**File:** `types.go` (lines 44–48)

**Problem:**  
The `InstanceStatus` type only defines three constants:

```go
const (
    InstanceCreating InstanceStatus = "creating"
    InstanceReady    InstanceStatus = "ready"
    InstanceDeleting InstanceStatus = "deleting"
)
```

The PRD (§3.1 schema) and the CHECK constraint in `0005_preview_instance.up.sql` define four valid statuses: `'creating','ready','updating','deleting','error'`. The `updating` and `error` statuses are referenced in the PRD (§5.2, §6) but have no Go constant. This will cause:

- Compile errors if any code path attempts to set status to `updating` or `error`.
- Silent bugs if status comparison uses string literals.

**Fix:**  
Add the missing constants:

```go
const (
    InstanceCreating InstanceStatus = "creating"
    InstanceReady    InstanceStatus = "ready"
    InstanceUpdating InstanceStatus = "updating"
    InstanceDeleting InstanceStatus = "deleting"
    InstanceError    InstanceStatus = "error"
)
```

---

### M2. `GraphNode.Type` and `GraphNode.ServiceType` both set to the same value

**File:** `instance.go` (lines ~170–175)

**Problem:**  
In `GetDependencyGraph`, both `Type` and `ServiceType` are populated from `n.ServiceType`:

```go
res.Nodes[i] = GraphNode{
    ID:          n.ID,
    Name:        n.Name,
    Type:        n.ServiceType,        // ← should be node category?
    ServiceType: n.ServiceType,        // ← service type (app/psql/redis)
}
```

The `Type` field appears intended for a node category (e.g., `"service"`, `"dependency"`) while `ServiceType` is the db service type. The duplication wastes a field and could confuse API consumers expecting distinct values.

**Fix:**  
Either remove `Type` if unused, or populate it with the actual node category (`"service"`).

---

### M3. `CreatePreviewRequest` validation tags unused

**File:** `preview.go` (lines 19–31)

**Problem:**  
`CreatePreviewRequest` has `validate:"required"` struct tags, but `CreatePreview` calls `c.Bind(&b)` — not `BindAndValidate`. The validation tags are dead code:

```go
type CreatePreviewRequest struct {
    ProjectID      uuid.UUID `json:"project_id" validate:"required"` // ← never checked
    Name           string    `json:"name" validate:"required"`       // ← never checked
    HeadBranch     string    `json:"head_branch" validate:"required"`// ← never checked
    GitSourceType  string    `json:"git_source_type" validate:"required,oneof=pr branch"` // ← never checked
    GitSourceValue string    `json:"git_source_value" validate:"required"` // ← never checked
}
```

Validation is deferred to `AssignCreatePreview` → `submit[T]`, which validates `CreatePreviewJobParams` instead. This means the HTTP handler accepts malformed input before the job submission fails.

**Fix:**  
Either switch to `BindAndValidate`, or remove the dead tags for clarity and add explicit handler-level validation.

---

### M4. Push handler silently suppresses DB error

**File:** `github.go` (lines ~508–510)

**Problem:**  
The push event handler silently discards the DB error from `GetAllAppServicesByRepo`:

```go
serviceIDs, err := q.GetAllAppServicesByRepo(h.qCtx, db.GetAllAppServicesByRepoParams{...})
if err != nil {
    return nil  // ← returns nil (HTTP 200) with no error logged
}
```

If the DB query fails, the webhook returns HTTP 200 OK with no indication of failure. GitHub will think processing succeeded and won't redeliver. This is a data-loss scenario — pushes that fail to retrieve services will never trigger a rebuild.

**Fix:**  
Log the error and return an appropriate HTTP 500 so GitHub retries delivery.

---

## 🔵 Low Severity

### L1. `ShutDownInstance` is a nil-return stub on wrong handler

**File:** `project.go` (lines ~235–241)

**Problem:**  
The stub `ShutDownInstance` lives on `ProjectHandler` with a route `POST /api/instance/shutdown`. The route is in the `instance` group but the handler is on `ProjectHandler`. This will be a nil-pointer panic if called because the route is not registered in `routes/core.go` (no `project.PUT("/shutdown", h.Project.ShutDownInstance)` line exists, and there's no instance-level shutdown route). However, since the route is never wired, the stub is unreachable dead code.

**Fix:**  
Either implement on `InstanceHandler` and register the route, or remove the stub.

---

### L2. `DeployPredefinedService` network creation is silently reattempted

**File:** `predef_worker.go` (line 30)

**Problem:**  
`DeployPredefinedService` calls `dockerClient.CreateNetwork(network)` which is idempotent (returns nil if network exists). This is correct but undocumented — the caller might expect an error if the network already exists. The PSQL and Redis handlers both call `GetInstanceNetwork` separately (line ~140 of psql_service.go, line ~126 of redis_service.go), then pass it to `DeployPredefinedService` which tries to create it again. Redundant but harmless.

**Suggestion:**  
Document that `CreateNetwork` is intentionally idempotent (or skip the call when the network is known to exist).

---

### L3. `ListPreviews` returns raw DB type in response

**File:** `preview.go` (line 113)

**Problem:**  
`ListPreviews` returns `types.Res[[]db.GetPreviewInstancesByProjectRow]` directly, leaking the sqlc-generated DB type to the API. This couples the API contract to the database schema. Other endpoints (e.g., `GetAllServices`) wrap their response in a handler-defined type (`GetAllServicesRes`).

```go
return c.JSON(http.StatusOK, types.Res[[]db.GetPreviewInstancesByProjectRow]{...})
```

**Fix:**  
Define a `ListPreviewsRes` struct and map from the DB type, following the pattern established by `GetAllServicesRes`.

---

### L4. `AppLogin` returns wrong `OrgId` in response

**File:** `auth.go` (line 190)

**Problem:**  
The `AppLogin` response sets `OrgId: u.ID` instead of `OrgId: org.ID`:

```go
return c.JSON(http.StatusOK, types.Res[AuthRes]{Message: "", Data: AuthRes{
    Name:    u.Name,
    Email:   u.Email,
    OrgId:   u.ID,  // ← should be org.ID (the Org ID, not user.ID)
    OrgName: u.OrgName,
}})
```

Compare with `AuthUser` (line 64) which correctly uses `org.ID`. This appears to be a pre-existing bug, not introduced by this PR, but it's in a file listed in the review scope (comment change).

**Note:** Marked as low because it's pre-existing and not part of the preview feature scope.

---

## 📋 Invariant Compliance Summary

| #   | Invariant                                                                                             | Status | Notes                                                 |
| --- | ----------------------------------------------------------------------------------------------------- | ------ | ----------------------------------------------------- |
| 1   | Webhook pull_request: opened/reopened → queue; synchronize → rebuild; closed → delete; all idempotent | ❌     | opened/reopened not idempotent; synchronize stub (C2) |
| 2   | issue_comment: HeadBranch="" must NOT fail validation                                                 | ❌     | Fails validation in submit[T] (C1)                    |
| 3   | GitHub manifest: contents=read, metadata=read, pull_requests=write, issues=write                      | ✅     | Correct in utils.go                                   |
| 4   | Domain changed from string to sql.NullString                                                          | ✅     | All paths use .String or .Valid                       |
| 5   | DeployPredefinedService handles network, volume, spec identically                                     | ✅     | Verified identical behavior                           |
| 6   | Preview routes protected by auth middleware                                                           | ✅     | Under `protected` group in routes/core.go             |
| 7   | ListPreviews must verify caller owns project                                                          | ❌     | No ownership check (H1)                               |
| 8   | CreatePreview passes GitSourceType + GitSourceValue to worker                                         | ✅     | Correctly forwarded                                   |
| 9   | InstanceStatus has all required values                                                                | ❌     | Missing `updating` and `error` (M1)                   |
| 10  | sqlc.yaml overrides match model.go                                                                    | ✅     | Overrides present for all new columns                 |

---

## Summary

| Severity    | Count | Key Action Required                                                                                                                         |
| ----------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| 🔴 Critical | 2     | Fix HeadBranch validation (C1); Implement RebuildPreviewOnPush (C2)                                                                         |
| 🟠 High     | 3     | Add auth check to ListPreviews (H1); Add validation to DeletePreview (H2); Make opened/reopened idempotent (H3)                             |
| 🟡 Medium   | 4     | Add missing InstanceStatus values (M1); Fix GraphNode type duplication (M2); Fix dead validation tags (M3); Log push handler DB error (M4)  |
| 🔵 Low      | 4     | Remove ShutDownInstance stub (L1); Document network idempotency (L2); Use typed response for ListPreviews (L3); Fix AppLogin OrgId bug (L4) |

**Total: 13 issues** (2 critical, 3 high, 4 medium, 4 low)
