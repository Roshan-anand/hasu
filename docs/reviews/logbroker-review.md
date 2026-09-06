# LogBroker Review — Architecture, Flaws & Improvements

## Architecture Summary

```
Deployment Workers (3)  ──PublishLog──>  pubData chan (cap 100)  ──>  Single LogBroker Worker  ──SSE──>  Browsers
                                                    │                         │
                                              endData chan (cap 100)     bufferPush (mem LogBuffer)
                                                    │                     ender → BadgerDB persist
                                              EndLogs calls
```

Single worker goroutine dispatches **all** log lines from **all** deployments serially. Every `PublishLog()` from any deployment worker enters the `pubData` channel and lands in the one `publisher()` → `bufferPush()` + SSE write path. Persistence to BadgerDB happens only at deployment end via `ender()`.

---

## Critical Flaws

### F1: Data Race on `l.subscribers`

`publisher()` and `ender()` iterate `l.subscribers` **without holding `l.mu`**. Meanwhile `Subscribe()` and `Unsubscribe()` (called from HTTP handler goroutines) mutate the map under `l.mu`. This is a concurrent map read/write race — Go's race detector will flag it, and it will cause panics under load.

**Location:** `worker.go:14` (`publisher` loop) and `worker.go:55` (`ender` loop)

### F2: SSE Write Blocks the Single Worker (Root Cause of Lag)

`sub.SSE.SendEvent()` calls `http.ResponseWriter.Write()` then `Flush()`. A slow or disconnected SSE client blocks the logbroker worker, which blocks `publisher()`, which blocks the `pubData` channel drain, which blocks **all** `PublishLog` callers across **all** deployments. The `scanAndPublish` loop in deployment workers then blocks on PTY reads, stalling Docker build output.

**Chain:** Slow browser → SSE.Write blocks → worker blocks → `pubData` fills (cap 100) → `PublishLog` blocks all deployment PTY readers → Docker output stalls.

### F3: `Stop()` Channel Close Race

`Stop()` calls `close(l.pubData)` and `close(l.endData)` **before** cancelling `egCtx` (which only happens on the timeout path). `PublishLog()` checks `<-l.egCtx.Done()` but `egCtx` is still alive. A deployment worker calling `PublishLog` during shutdown races with the `close()` and can send on a closed channel → **panic**.

**Fix:** Cancel `egCtx` first, then close channels after `eg.Wait()` returns.

### F4: Single Worker Bottleneck

All deployments (up to 3 workers, each streaming PTY output line-by-line) funnel through one goroutine. Even with fast SSE clients, serial processing of every line from every deployment is a hard throughput cap. Each line does O(S) subscriber iteration, mutex-locked buffer append, and SSE marshalling.

### F5: No Incremental Persistence — Crash = Data Loss

`LogBuffer` is purely in-memory. Only written to BadgerDB in `ender()` when the deployment finishes (success/error/cancel). Server crash or kill mid-deployment loses all accumulated logs.

---

## Severity-2 Issues

### I1: Unbounded Memory (no log buffer cap)

`LogBuffer` is `map[uuid.UUID][]string` with no per-deployment limit. A deployment producing 100K+ lines of Docker output holds all of them in memory until `EndLogs()`. An attacker or runaway build could OOM the server.

### I2: `pubData` Channel Capacity Too Low (100)

During burst PTY output (e.g. `npm install` spitting hundreds of lines), 3+ deployment workers rapidly fill the 100-slot channel and then all block, completely serializing their independent PTY reads through the single worker.

### I3: `ender()` Ignores Cancellation Context

```go
l.quries.UpdateDeploymentStatus(context.Background(), ...)
```

Uses `context.Background()` instead of the service's `egCtx`. A slow DB write during shutdown cannot be interrupted.

### I4: No Subscriber Heartbeat

Long-lived SSE connections have no keep-alive mechanism. Proxies/load balancers may close idle connections. Clients can't distinguish "no logs" from "connection dead".

### I5: `bufferGet` Returns Nil Slice for New Deployments

`bufferGet` returns `l.logBuffer[dID]` which for a never-before-seen deployment ID returns a nil slice. `ender()` then does `logs = append(logs, d.Message)` — works fine, but `publisher()` marshals nil to `"null"` JSON and sends it as `"logs"` event to new subscribers. Frontend receives `null` instead of `[]`.

### I6: `ender()` Error Message Never Sent to Clients if Empty

```go
if d.Status == types.DeploymentError {
    if d.Message == "" {
        d.Message = "something went wrong !!"
    }
}
```

This mutates `d.Message` **after** `logs = append(logs, d.Message)` was already computed. Persisted logs get the old empty message. Only the final SSE push gets the default message. Persistence and client delivery diverge.

---

## Severity-3 Issues

### N1: `ender()` Uses `l.Unsubscribe` While Iterating Map

`ender()` iterates `l.subscribers` and calls `l.Unsubscribe(userID)` which deletes from the same map under `l.mu` — but `ender()` doesn't hold `l.mu` (see F1). Even after fixing the race, deleting while iterating a Go map is safe, but the pattern is fragile and should use a "collect keys, then delete" approach.

### N2: No Log Compression / Truncation

Raw Docker build output lines are stored as-is. No dedup of repeated lines, no ANSI escape stripping, no size-based truncation. BadgerDB keys use 6-digit zero-padded index — caps at 999,999 lines per deployment (unlikely but present).

### N3: `scanAndPublish` Swallows Scanner Errors

`fmt.Println("scan error :", err)` — errors are logged to stdout but never surfaced to the caller or the user. If the PTY breaks mid-stream, the deployment continues silently with partial logs.

---

## Recommended Improvements

### Fix F1 (Race): Hold `l.mu` During Subscriber Iteration

```go
func (l *LogBrokerService) publisher(d *PubData) {
    l.mu.Lock()
    subs := make([]*Subscriber, 0, len(l.subscribers))
    for _, sub := range l.subscribers {
        subs = append(subs, sub)
    }
    l.mu.Unlock()

    for _, sub := range subs {
        if sub.DeploymentID == d.ID {
            // ...same logic...
        }
    }
    l.bufferPush(d.ID, d.Msg)
}
```

### Fix F2 (Blocking SSE): Fan-Out SSE Writes to Per-Subscriber Goroutines

Give each subscriber a buffered channel (`chan []byte, cap N`). `publisher()` enqueues the log line and returns immediately. A dedicated subscriber goroutine drains the channel and does SSE writes. Slow clients only block their own channel, never the main worker.

```
publisher() → sub.channel (non-blocking send or drop) → subscriber goroutine → SSE.Write+Flush
```

Optionally: if subscriber channel is full, drop the log line (or close the subscriber). **Never block the main worker on client I/O.**

### Fix F3 (Stop Race): Cancel Context Before Closing Channels

```go
func (l *LogBrokerService) Stop(ctx context.Context) error {
    l.cancel()               // signal PublishLog/EndLogs to stop
    <-l.eg.Wait()            // wait for worker to drain
    close(l.pubData)
    close(l.endData)
    return nil
}
```

### Fix F4 (Worker Pool): Multiple Publisher Workers

Replace single `l.eg.Go(worker)` with N workers all selecting on the same channels. Go channels are MPMC-safe. Each deployment deploymentID's logs stay ordered within that deployment (since all lines for one deployment originate from one PTY goroutine, and channels preserve order for single-producer). Workers can still share the mutex for buffer and subscriber operations.

```go
for i := 0; i < runtime.NumCPU(); i++ {
    l.eg.Go(func() error { return l.worker(l.egCtx) })
}
```

### Fix F5 (Crash Safety): Periodic/Incremental BadgerDB Flush

Option A: Flush every N log lines per deployment.  
Option B: Flush on a timer (every 5s).  
Option C (simpler): Flush the buffer atomically **before** each `publisher()` call — this doubles writes but guarantees at-most-one-line loss.

### I1 Fix: Ring Buffer Per Deployment

Cap `LogBuffer` at a max line count (e.g. 10,000). Beyond that, drop oldest lines or flush to BadgerDB.

### I2 Fix: Increase `pubData` to 10,000 or Remove Channel Entirely

With fan-out subscriber goroutines (F2 fix), the worker drains fast enough that 1000–10,000 is reasonable. Or: replace the channel with a lock-free ring buffer to eliminate the goroutine-crossing overhead entirely.

### I4 Fix: Periodic SSE Comment Heartbeat

In subscriber goroutine: every 15s, send `: heartbeat\n\n` (comment event, ignored by browsers, resets proxy timeouts).

### I5 Fix: Return Empty Slice Instead of Nil

```go
func (l *LogBrokerService) bufferGet(dID uuid.UUID) []string {
    l.mu.Lock()
    defer l.mu.Unlock()
    logs := l.logBuffer[dID]
    if logs == nil {
        return []string{}
    }
    return logs
}
```

### I6 Fix: Set Default Message Before Appending to Logs

```go
if d.Status == types.DeploymentError && d.Message == "" {
    d.Message = "something went wrong !!"
}
logs := l.bufferGet(dID)
logs = append(logs, d.Message)
```

### N1 Fix: Collect Keys Before Unsubscribing

```go
toRemove := make([]uuid.UUID, 0)
for userID, sub := range l.subscribers {
    if sub.DeploymentID == dID {
        sub.SSE.SendEvent("log", []byte(d.Message))
        toRemove = append(toRemove, userID)
    }
}
for _, userID := range toRemove {
    l.Unsubscribe(userID)
}
```

---

## Priority Roadmap

| Priority | What | Why |
|----------|------|-----|
| **P0** | Fix F1 (race) | Will panic under load, data corruption |
| **P0** | Fix F2 (blocking SSE) | Root cause of user-visible lag |
| **P0** | Fix F3 (Stop race) | Will panic on graceful shutdown |
| **P1** | Fix F4 (worker pool) | Throughput scaling for multi-deployment |
| **P1** | Fix F5 (incremental persist) | Crash safety |
| **P2** | Fix I1 (memory cap) | OOM prevention |
| **P2** | Fix I2 (channel cap) | Burst throughput |
| **P2** | Fix I6 (ender message order) | Correctness |
| **P3** | Fix I4 (heartbeat) | Connection reliability |
| **P3** | Fix I5 (nil slice) | Frontend JSON contract |
| **P3** | Fix I3 (context in ender) | Clean shutdown |
| **P3** | Fix N1-N3 (code hygiene) | Maintainability |
