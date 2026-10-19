# ADR-0003: no ORM, no third-party router, CSRF from stdlib

**Date:** 2026-10-14.
**Status:** accepted.

This ADR pins three related "no"s.

## 1. No ORM

### Context

It is common in Go to wrap relational DBs in an ORM. Best known: `gorm`
(active record), `ent` (Facebook), `sqlx` (a thin layer over `database/sql`).

### Decision

`database/sql` + hand-written parameterized queries (`?`-placeholders).

### Why

| Argument | Detail |
|---|---|
| **Security** | Easier to read SQL and audit it by eye — no abstractions hiding unsafe construction. |
| **Size** | No runtime model metadata. |
| **Cold start** | `ent` and `gorm` load reflection code at startup; `database/sql` does not. |
| **Predictability** | One `ExecContext(ctx, query, args...)` — no "lazy loading", no "this becomes an N+1 under the hood". |
| **DB migration** | When moving (SQLite → Postgres) SQL is almost 1:1 portable. ORM-specific constructs (`gorm.Model`, `ent.Schema`) — not so much. |

### When to revisit

- If we cross 50 tables and SQL starts looking like a join network — at that
  point [sqlc](https://github.com/sqlc-dev/sqlc) gives type-safe SQL without
  runtime overhead.
- If we need complex workflows (state machines, soft-deletes, audit trails) —
  revisit.

## 2. No third-party router (chi/gin/echo/fiber)

### Context

Until Go 1.22, `http.ServeMux` did not support patterns (`/users/{id}`), so
people reached for `gorilla/mux`, `chi`, or a framework like `gin`/`echo`.

### Decision

`net/http` + `http.ServeMux` from Go 1.22+.

### Why

- **Patterns are now in stdlib:** `mux.HandleFunc("GET /users/{id}", ...)` →
  `r.PathValue("id")`.
- **Method matching:** `mux.HandleFunc("POST /users", ...)` separate from
  `GET /users/{id}`.
- **Root-only:** `mux.HandleFunc("GET /{$}", ...)` so `/` does not also
  match `/foo`.
- **Trailing slash:** prefix vs exact vs sub-tree — all in one syntax.
- **Middleware works out of the box:** `mux.Handle("POST /login", wrap(handler))`.
- **Zero dependencies + CVE audit.**

### When to revisit

- If we need route precedence / overlap that ServeMux does not provide — it
  shows up clearly (e.g. `/users/me` vs `/users/{id}`). Solved by
  `mux.HandleFunc("GET /users/me", ...)` **before** `/users/{id}` (order matters).

## 3. CSRF from stdlib

### Context

The classic Go CSRF middleware is `gorilla/csrf` (csrf token as a hidden form field).
In Go 1.25+ `http.NewCrossOriginProtection()` appeared.

### Decision

`http.NewCrossOriginProtection()`, wrapping the whole mux.

### Why

- **Less surface for bugs.** A hand-rolled CSRF middleware breaks easily.
  The stdlib one went through an RFC discussion and a thorough security review.
- **No tokens in forms** — the user sees a clean form. The middleware works
  at the header level (`Sec-Fetch-Site`, `Origin`).
- **No dependency** — `net/http` is already in Go.
- **Less code** — 2 lines in `internal/server/server.go`.

### Behavior

- `GET` / `HEAD` / `OPTIONS` — always passed through.
- `POST` / `PUT` / `PATCH` / `DELETE` with `Sec-Fetch-Site: same-origin` or
  `none` — passed through.
- `Origin` matching `Host` — passed through.
- Origin in `AddTrustedOrigin(...)` — passed through.
- Otherwise — `403 Forbidden`.
- Requests **without** `Origin` and `Sec-Fetch-Site` (curl, many API clients) —
  passed through.

### When to revisit

- If we add a real form (not an SPA) where we need to approve requests **without**
  browser headers — add a hidden field with a token and middleware that
  validates the field AND a cookie. But that brings back the gorilla/csrf
  complexity. The alternative — to require `Origin` via the reverse proxy —
  is what we already do.

## Decision dependencies

- [ADR-0001](0001-stdlib-first.md) — the general "minimum deps" principle.
- [ADR-0002](0002-argon2id-and-cookie-sessions.md) — why Argon2id and cookie sessions.
