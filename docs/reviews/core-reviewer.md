# Rebuild Cancellation & Delayed Promotion — Code Review

## Review Summary

**Score: 6/10** — The registry design and newest-wins semantics are sound, but there are critical startup-crash bugs, a double-EndLogs issue, a race window in CancelDeployment, context leaks, and a missing registry cleanup in the cancel path. Fix the CRITICAL items before deployment.

---

## CRITICAL — Will Crash or Corrupt

### 5. Context Leak: CleanupRebuild doesn't call cancel

**Files:** `core.go` (`CleanupRebuild`, line ~440) + `core.go` (`RegisterRebuild`, line ~390)

`RegisterRebuild` creates `ctx, c := context.WithCancel(parentCtx)`. When a rebuild completes normally and `CleanupRebuild` is called, it removes the entry from the map but **never calls `c()`**. 

In Go, each `context.WithCancel` starts an internal goroutine that waits for the parent to be canceled. Without calling `cancel()`, this goroutine leaks until `parentCtx` (`d.egCtx`, i.e., server shutdown) is canceled.

**Impact:** Memory/goroutine leak proportional to the number of completed rebuilds. Not catastrophic but wasteful.

**Fix:** Add `entry.cancel()` in `CleanupRebuild` (and also in `CancelRebuild` — which already does call `entry.cancel()`). Note: `CancelRebuild` already calls `entry.cancel()` but `CleanupRebuild` does not. Be consistent:

```go
func (d *DeploymentService) CleanupRebuild(serviceID uuid.UUID, jobID int64) bool {
    d.rebuildMu.Lock()
    defer d.rebuildMu.Unlock()
    entry, ok := d.rebuilds[serviceID]
    if !ok || entry.jobID != jobID {
        return false
    }
    delete(d.rebuilds, serviceID)
    if entry.cancel != nil {
        entry.cancel()
    }
    return true
}
```

---
### 10. PromoteDeploymentToCurrent uses context.Background() for transaction

**File:** `app_worker.go` (`PromoteDeploymentToCurrent`, line ~505)

```go
tx, err := d.db.Pool.BeginTx(context.Background(), nil)
```

The transaction uses `context.Background()` instead of a caller-provided context. During shutdown, this transaction cannot be interrupted. If the DB is slow or locked, shutdown could hang for the full transaction timeout. Consider passing a context with a reasonable deadline from the caller.

---

### 12. `RollbackAppService` handler still has TODO about checking for active rebuild

**File:** `handlers/deployment.go` (line ~216)

The comment says:
```go
// TODO : current - checks if ongoing rebuild work and stops the Rollback
// update it to - cancle the rebuild work and do rollback (do some validation)
```

While `HasActiveRebuild` now gates rollback (line ~221), the TODO mentions eventually *canceling* the rebuild and proceeding with rollback. This is an intentional design choice, not a bug — just noting the TODO is stale.

### 13. `runDeploymentPipeline` doesn't check `ctx.Err()` for cancellation

**File:** `app_worker.go` (`runDeploymentPipeline`)

The old deploy pipeline for initial deployments doesn't check context cancellation at all. Unlike `RunRebuildPipeline` which has cancellation checks throughout, `runDeploymentPipeline` will run to completion even if the service is shutting down. This is pre-existing but worth noting since the PRD emphasizes context propagation.

### 14. `CancelDeployment` error wrapping hides `sql.ErrNoRows`

**File:** `core.go` (`CancelDeployment`, line ~610)

```go
deployment, err := d.db.Queries.GetDeployment(ctx, deploymentID)
if err != nil {
    return fmt.Errorf("deployment not found: %w", err)
}
```

If `GetDeployment` returns `sql.ErrNoRows`, the error message says "deployment not found" which is accurate. But if it returns a DB connection error, the wrapper says "deployment not found" which is misleading. Consider distinguishing:

```go
if errors.Is(err, sql.ErrNoRows) {
    return fmt.Errorf("deployment not found")
}
return fmt.Errorf("failed to fetch deployment: %w", err)
```

### 15. Race: `CancelDeployment` Finds DB Row But Registry `deploymentID` Is nil
**Files:** `apps/server/internal/jobs/deployment/core.go:615-625` + `app_worker.go:189-192`
Worker creates deployment DB row → then calls `SetRebuildDeploymentID`. Between those two points, `CancelDeployment` finds the DB row but `entry.deploymentID` is nil, returning `"deployment is not owned by an active rebuild"`.
**Fix:** Move deployment UUID generation before DB insert, call `SetRebuildDeploymentID` before `CreateDeployment`. If DB insert fails, remove from registry.
