# Handler Diff Review — Rebuild Cancellation & Delayed Promotion

## Files Reviewed
- `apps/server/internal/handlers/deployment.go`  (+56/-)
- `apps/server/internal/handlers/github.go`      (+11/-)
- `apps/server/internal/handlers/project.go`      (+9/-)

---

## 🔴 CRITICAL: SSE `for-range` over `Done()` never executes loop body

**File:** `deployment.go`, `SubscribeServiceDeploymentLogs`, lines ~146–152

```go
// BEFORE (correct)
for {
    select {
    case <-c.Request().Context().Done():
        log.Printf("SSE client disconnected, ip: %v", c.RealIP())
        l.Unsubscribe(userID)
        return nil
    }
}

// AFTER (broken)
for _ = range c.Request().Context().Done() {
    log.Printf("SSE client disconnected, ip: %v", c.RealIP())
    l.Unsubscribe(userID)
    return nil
}
return nil
```

**Why it's broken:** `c.Request().Context().Done()` returns a `<-chan struct{}` that is **closed** (never sent on) when the context cancels. Go's `range` over a channel iterates received values until the channel is closed. Since no values are ever sent on the `Done()` channel, the loop body **never executes**.

**Concrete effect:**
1. The handler *does* block correctly while the client is connected (range blocks on open channel).
2. When the client disconnects, Done() is closed → `range` exits immediately, skipping the body.
3. `l.Unsubscribe(userID)` is **never called** → subscriber leak in the log broker.
4. The `log.Printf` disconnect message is never emitted.
5. Execution falls through to the dead `return nil` on the line after the loop.

**Fix:** Replace the for-range with a plain blocking receive:

```go
<-c.Request().Context().Done()
log.Printf("SSE client disconnected, ip: %v", c.RealIP())
l.Unsubscribe(userID)
return nil
```

The outer `for {}` + `select` was already redundant — a single-case select blocks identically to a plain receive. The `for` was dead code in the original too.

---

## 🔴 HIGH: `MergeDependencyEnv` returns `nil` on error, wiping all env vars

**File:** `apps/server/internal/jobs/deployment/app_worker_utils.go`, line ~347

```go
func MergeDependencyEnv(q *db.Queries, sourceServiceID uuid.UUID, manualEnv []string) []string {
    rows, err := q.ResolveDependencyEnv(context.Background(), sourceServiceID)
    if err != nil {
        return nil  // ← BUG: discards caller's env vars
    }
    for _, row := range rows {
        manualEnv = append(manualEnv, fmt.Sprintf("%s=%s", row.EnvKey, row.ResolvedValue))
    }
    return manualEnv
}
```

**Call site in `RedeployAppService` (deployment.go, line ~307):**
```go
env.Env = deployjob.MergeDependencyEnv(q, b.ServiceID, env.Env)
```

If the DB query fails (e.g., transient connection error), `MergeDependencyEnv` returns `nil`. This **silently replaces all manually configured env vars with `nil`**, and the redeploy runs with zero environment variables.

**Fix:**
```go
if err != nil {
    return manualEnv  // return unmodified on error
}
```

---

## 🔴 HIGH: `CancelDeployment` returns 500 for all semantic errors

**File:** `deployment.go`, `CancelDeployment`, lines ~277–289

```go
if err := h.Server.Services.Deployment.CancelDeployment(b.DeploymentID); err != nil {
    return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to cancel deployment"})
}
```

The service (`core.go:592–641`) returns **distinct, meaningful errors**:

| Service error | Should be | Currently returns |
|---|---|---|
| `"deployment not found: …"` | **404** | 500 |
| `"cannot cancel finished deployment"` | **409** | 500 |
| `"cannot cancel deployment with status X"` | **400** | 500 |
| `"deployment is not owned by an active rebuild"` | **409** | 500 |
| `"failed to update deployment status: …"` | 500 ✅ | 500 |

**Violates invariant #1** from the PRD. The frontend has no way to distinguish "this deployment doesn't exist" from "server crashed."

**Fix:** The service should export sentinel errors, and the handler should map them:

```go
// In core.go — add sentinel errors
var (
    ErrDeploymentNotFound      = errors.New("deployment not found")
    ErrDeploymentFinished      = errors.New("cannot cancel finished deployment")
    ErrDeploymentNotOwned      = errors.New("deployment is not owned by an active rebuild")
)

// In handler
if err != nil {
    switch {
    case errors.Is(err, deployjob.ErrDeploymentNotFound):
        return c.JSON(http.StatusNotFound, types.Res[struct{}]{Message: err.Error()})
    case errors.Is(err, deployjob.ErrDeploymentFinished),
         errors.Is(err, deployjob.ErrDeploymentNotOwned):
        return c.JSON(http.StatusConflict, types.Res[struct{}]{Message: err.Error()})
    default:
        return c.JSON(http.StatusInternalServerError, types.Res[struct{}]{Message: "failed to cancel deployment"})
    }
}
```

Also: the error message in the 500 response is generic `"failed to cancel deployment"`. Consider including the actual error (or at least logging it server-side).

---

## 🟡 MEDIUM: Webhook `AssignRebuild` failure silently swallowed — no log

**File:** `github.go`, `GithubWebhook`, lines ~491–498

```go
if _, _, _, err := h.Server.Services.Deployment.AssignRebuild(context.Background(), &deployjob.RebuildServiceParams{
    ServiceID:  sID,
    CommitHash: pushEvent.GetAfter(),
    CommitMsg:  pushEvent.GetHeadCommit().GetMessage(),
    Source:     "webhook",
}); err != nil {
    // Failure for one service should not block others; the error is already
    // logged by the deployment service.
    continue
}
```

**The comment is wrong.** The deployment service's `submit()` function does **not** log errors — it returns them. `AssignRebuild` calls `submit()` and propagates the error without logging. The error is **truly silent**.

**Fix:** Add at minimum a `log.Printf`:

```go
if _, _, _, err := h.Server.Services.Deployment.AssignRebuild(...); err != nil {
    log.Printf("webhook: assign rebuild failed for service %s: %v", sID, err)
    continue
}
```

---

## 🟡 LOW: Debug `fmt.Println` left in webhook handler

**File:** `github.go`, line ~477

```go
fmt.Println("recived commit push for :", repo, branch)
```

- Typo: `recived` → `received`
- Uses `fmt.Println` instead of structured `log.Printf`
- This is a debug print that should either be removed or converted to proper logging

---

## 🟡 LOW: Rollback TODO has trailing whitespace

**File:** `deployment.go`, lines ~200–201

```go
// TODO : current - checks if ongoing rebuild work and stops the Rollback
// update it to - cancle the rebuild work and do rollback (do some validation)
	
```

- Typo: `cancle` → `cancel`
- The blank line after the comment has trailing spaces (visible in diff)
- The TODO suggests future work to cancel rebuild then rollback; the current `HasActiveRebuild` check is correct as a first pass

---

## 🟢 OK: `HasActiveRebuild` guards on Redeploy and Rollback

Both checks are correct and non-racy for the current in-memory registry:

```go
// Redeploy (L303-305)
if h.Server.Services.Deployment.HasActiveRebuild(b.ServiceID) {
    return c.JSON(http.StatusConflict, ...)
}

// Rollback (L200-202)
if h.Server.Services.Deployment.HasActiveRebuild(b.ServiceID) {
    return c.JSON(http.StatusConflict, ...)
}
```

The read-lock in `HasActiveRebuild` (core.go:580–584) is adequate for an advisory check. Rebuild handler intentionally does **not** check — `RegisterRebuild` supersedes in-flight rebuilds, which is correct behavior.

---

## 🟢 OK: `sql.ErrNoRows` check in Redeploy

```go
if err == sql.ErrNoRows {
    return c.JSON(http.StatusNotFound, ...)
}
```

Clean. `GetAppServiceForRedeploy` uses `:one` sqlc query, so `sql.ErrNoRows` is the right sentinel. Wraps correctly — sqlc returns this unwrapped.

---

## 🟢 OK: Image/swarm 404 in Redeploy

Previously both returned 500; now return 404. Correct: missing image/swarm is a caller error (stale deployment record), not a server error.

---

## 🟢 OK: Project handler `ShutDownInstance` removal

Removed dead stub. Added a useful TODO on `DeleteProject` about preview instances. Clean.

---

## Summary

| Severity | Issue | File |
|---|---|---|
| 🔴 Critical | SSE `for-range` over `Done()` never fires body — `Unsubscribe` leak | deployment.go |
| 🔴 High | `MergeDependencyEnv` returns `nil` on error, wipes env vars | app_worker_utils.go |
| 🔴 High | `CancelDeployment` returns 500 for 404/409 semantic errors | deployment.go |
| 🟡 Medium | Webhook `AssignRebuild` error silently swallowed | github.go |
| 🟡 Low | Debug `fmt.Println` with typo | github.go |
| 🟡 Low | TODO typos and trailing whitespace | deployment.go |

**Blockers for merge:** The SSE bug (🔴) and the `MergeDependencyEnv` nil-return (🔴) should be fixed before deploy. The CancelDeployment status-code issue (🔴) should be fixed before the endpoint is exposed to the frontend.
