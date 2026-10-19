# Architecture

This document describes **the "why"**, **what is locked in**, and **what is flexible**.
For operational details — [deploy.md](deploy.md), [reference.md](reference.md),
[recipes/](recipes/). For the rationale — [adr/](adr/).

## Principles

1. **Stdlib-first.** The Go 1.27 standard library covers 95% of needs.
   An external dependency only when stdlib is provably weaker. See [ADR-0001](adr/0001-stdlib-first.md).
2. **Secure by default.** CSP, CSRF, HSTS, secure cookies, Argon2id — wired once.
   They cannot be turned off without a deliberate decision and a justified PR.
3. **Observability built-in.** Request ID, structured logging, `/healthz`, `/readyz`.
4. **Graceful everything.** SIGTERM → drain → exit; SIGHUP → reload rate-limit and log level.
5. **Static binary, no runtime.** `CGO_ENABLED=0`; one 13 MB binary; runs anywhere.

## What is locked in (do not change without discussion)

| Decision | Why | Where it lives |
|---|---|---|
| **Go 1.27+** | requires stdlib `uuid` and `http.NewCrossOriginProtection` | `go.mod` |
| **3 deps** (`x/crypto`, `x/time`, `sqlite`) | each justified; see [ADR-0001] | `go.mod` |
| **`net/http` + `http.ServeMux`** | pattern routing 1.22+ covers chi/echo needs | `internal/server/server.go` |
| **`database/sql`, no ORM** | parameterized `?`-queries are safer and more transparent | `internal/store/*.go` |
| **`html/template`** | auto-escaping for XSS | `internal/render/render.go` |
| **Argon2id** with `m=64 MiB, t=3, p=4` | OWASP 2024, with DoS limits | [ADR-0002](adr/0002-argon2id-and-cookie-sessions.md) |
| **Cookie sessions in DB**, not JWT | trivial invalidation on logout / password change | [ADR-0002] |
| **`http.NewCrossOriginProtection`** | hand-rolled CSRF tends to be wrong | [ADR-0003](adr/0003-no-orm-no-router.md) |
| **CSP with per-request nonce** | only safe way to keep inline-script | `internal/middleware/headers.go` |
| **`SetMaxOpenConns(1)` for SQLite** | otherwise `:memory:` in tests diverges | `internal/store/store.go` |
| **Middleware chain:** `Recovery → RequestID → SecurityHeaders → MaxBodySize → Logging → CSRF → mux` | see below | `internal/server/server.go` |

Any change to these requires an ADR or an issue first.

## What is flexible (typical places to change)

| Layer | Where | When to edit |
|---|---|---|
| Templates | `web/templates/{layouts,partials,pages}/` | adding a page, redesign |
| Handlers | `internal/handlers/` | new page/endpoint |
| Store methods and migrations | `internal/store/` + `internal/store/migrations/` | new entity |
| Config | `internal/config/config.go` + `.env.example` | new setting |
| Logging | `internal/middleware/logging.go` | adding fields to the log line |
| Error tracking | `internal/logging/logging.go` | swapping the `slog` handler |

## Request flow

One middleware wrapper, one place to change. Current order
(`internal/server/server.go`, numbered outside→inside):

```
Recovery → RequestID → SecurityHeaders → MaxBodySize → Logging → CSRF → mux
```

Earlier the chain was `Logging → RequestID → ...`, which left Logging unable to
see the request ID through context (Logging was outside RequestID, and
`r.WithContext()` creates a new request). Details are in git history.

### Purpose of each layer

| Layer | What it does | When to edit |
|---|---|---|
| Recovery | catches panic, logs with stack + request_id, returns 500 | almost never (already written) |
| RequestID | `X-Request-Id` from header or stdlib `uuid.New()` | only if you want a different ID source |
| SecurityHeaders | CSP nonce + HSTS + X-Content-Type-Options + Permissions-Policy + Referrer-Policy | new policy headers |
| MaxBodySize | limit body size (1 MB) | raise limit for uploads |
| Logging | method/path/status/duration/ID | new log fields |
| CSRF | blocks unsafe cross-origin POST/PUT/DELETE/PATCH | trusted origins via `AddTrustedOrigin` |
| mux | `net/http` dispatcher | registering routes |

Handlers sit behind `mux`. If a handler wants to read context values
(`CSP nonce`, `request_id`, `user_id`) — it pulls them via `middleware.*From(r.Context())`.

## Why exactly so

### `http.NewCrossOriginProtection` instead of a hand-rolled CSRF

Every hand-rolled CSRF middleware I have seen had a bypass.
`http.CrossOriginProtection` (Go 1.25+, stdlib) does exactly what you need:
blocks unsafe cross-origin methods based on `Sec-Fetch-Site` and `Origin`. Ten
lines of integration in `server.go` — no tokens, no hidden fields in forms.

### Cookie sessions in DB, not JWT

JWTs cannot be invalidated on the server in principle: the token is valid until
expiry regardless of what the server thinks. Logout is a "polite request" to the
client.

A cookie session in DB is `cookie = session_id`. The server alone decides
whether the session is valid, and can kill it at any moment (`auth.DeleteSession`).

### Argon2id with parameter validation

OWASP 2024 says `m ≥ 64 MiB, t ≥ 3, p ≥ 1`. We use
`m=64 MiB, t=3, p=4` ([`auth.DefaultParams`](../internal/auth/password.go)).

But there is a risk: if an attacker controls a hash in DB (for example via SQLi
in profile editing), they could submit a hash with `m=1GB`. Argon2 would eat
all memory and take the service down.

The mitigation is in [`auth.VerifyPassword`](../internal/auth/password.go):
the limits (`maxMemory=256 MiB`, `maxIterations=10`, `maxParallelism=16`) are
checked **before** Argon2 runs. The test `TestVerifyRejectsHugeParams` pins
this contract.

### CSP with per-request nonce, not `unsafe-inline`

The CSP spec requires a unique nonce per response. If you reuse one nonce for
the whole application, an XSS vector in one response lets an attacker inject
`<script nonce="...">` in another response.

`middleware.SecurityHeaders` generates the nonce **before** setting the CSP
header; the handler reads it via [`middleware.CSPNonceFrom(r.Context())`](../internal/middleware/headers.go)
and inserts it into `<script nonce="...">`. Header and body are in sync.

### `SetMaxOpenConns(1)` for SQLite

SQLite is single-threaded for writes. If you open 2 connections to the same
file, one will see data the other has not committed yet. With `:memory:` it is
worse: every connection gets **its own** database, migrations "cannot see"
queries.

`store.Open` sets `SetMaxOpenConns(1)` for all cases, not just tests. This
hurts under load (no parallelism), but SQLite is already single-threaded for
writes, so we lose nothing — we just simplify reasoning.

### `base.Clone()` for each page

`html/template` stores all templates in a shared namespace. If you load
`pages/{home,about}.html` into a single `*template.Template`, both pages
overwrite each other's `{{ define "content" }}`.

The fix is in [`render.init`](../internal/render/render.go): `base.Clone()`
for each page, then `ParseFS` with a single page file. You get one namespace
per page, with shared partials/layout.

## Anti-patterns (full list — [anti-patterns.md](anti-patterns.md))

The most critical:

1. `template.HTML(...)` on user input → XSS.
2. Forgotten `base.Clone()` → all pages render the same content.
3. Hand-rolled CSRF → ~90% chance of a hole.
4. `github.com/google/uuid` → stdlib version exists, conflict with the architecture.
5. `:memory:` SQLite without `SetMaxOpenConns(1)` → migrations in one DB, tests in another.
6. No per-request nonce → either `unsafe-inline` (XSS) or broken JS.

## When **this template is not a fit**

| Need | Choose |
|---|---|
| Heavy dynamic UI, drag-and-drop, autosave | React/Vue/Svelte SPA + a separate Go API |
| Real-time collaboration (Figma-like) | Liveblocks / Partykit; or Y.js + WebSocket |
| Complex JOINs over 5+ tables | Postgres + sqlc (even in Go) |
| GraphQL API | gqlgen — already a 4th dependency, does not fit |
| ML/AI in the backend | Python microservice |
| A large team on a shared codebase | NestJS / Next — more out-of-the-box structure |

The Go-vs-JS tradeoff is not the topic of this document; see the general Go
ecosystem consensus.

## What is intentionally NOT in the template

- **Email verification, password recovery.** The template is minimal; either would require an SMTP client.
- **OAuth, 2FA, passkeys.** Easy to extend via `internal/auth/`, but not in the base set.
- **CSRF tokens in forms.** Not needed — `http.CrossOriginProtection` does the work without them.
- **HTTP / JSON API.** Not part of MVP; see [recipes/add-api.md](recipes/add-api.md) when you need it.
- **WebSocket / SSE.** Same: [recipes/add-realtime.md](recipes/add-realtime.md).
- **Background jobs.** Same: [recipes/add-job.md](recipes/add-job.md).
- **User registration.** Deliberately absent — for admin tools users are created by hand; for a public site a handler is added.

## Where to read next

- [deploy.md](deploy.md) — where this all runs
- [reference.md](reference.md) — env / routes / structure
- [recipes/](recipes/) — how to add a page / entity / API
- [anti-patterns.md](anti-patterns.md) — what not to do
- [adr/](adr/) — why exactly these decisions
