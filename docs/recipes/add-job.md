# Recipe: background jobs

**Status:** placeholder.

The template is deliberately single-process. If you need background jobs, four
options from simple to complex:

### Option 1. A goroutine in `main.go`

For 1–2 jobs without retries and without persistence:

```go
go func() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for range ticker.C {
        _ = doCleanupWork(ctx, s) // catch errors inside
    }
}()
```

Good for cron-like jobs that survive restart-from-zero and do not run for a long time.

### Option 2. Pull queue in SQLite

`internal/jobs/` — table `pending_jobs` (id, run_after, payload, attempts).
A dedicated goroutine `for { claim-and-run }`. Simple retry logic.

Good for: email notifications, per-request background cleanup, analytics.

### Option 3. Separate binary + supervisord

`cmd/worker/main.go` — a separate executable; runs next to the main server.
Shares `internal/store/`, `internal/auth/`, etc.

Good for: heavy jobs that should not compete with the web server for CPU.

### Option 4. External queue (would add a 4th dep to the template)

Redis + a third-party SDK or Postgres functions. **Adds 1+ deps.**
See [ADR-0001](../adr/0001-stdlib-first.md).

The recipe will appear the first time a job actually has to be implemented.
