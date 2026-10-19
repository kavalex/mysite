# ADR-0001: stdlib-first (3 dependencies)

**Date:** 2026-10-14.
**Status:** accepted.

## Context

The Go ecosystem offers dozens of packages for a minimal web application:

- routers: `chi`, `gin`, `echo`, `fiber`, `gorilla/mux`;
- ORMs: `gorm`, `ent`, `sqlx`, `sqlc`;
- sessions: `gorilla/sessions`, `alexedwards/scs`;
- CSRF: `gorilla/csrf`, `justinas/nosurf`;
- validators: `go-playground/validator`, `asaskevich/govalidator`;
- middleware helpers: `urfave/negroni`, `gofiber/fiber`, `justinas/alice`.

Each one brings:

1. **attack surface** — new surface for CVE audit;
2. **lock-in** — refactoring on a router/ORM swap means rewriting half the code;
3. **drift** — dependency versions diverge; CI breaks "for no reason";
4. **onboarding cost** — a new developer has to learn the framework, not stdlib.

On the other hand, the Go 1.27 standard library covers 95% of needs:
`net/http`, `html/template`, `database/sql`, `crypto/rand`/`crypto/subtle`,
`log/slog`, `crypto/sha256` (but **not** Argon2 — that hole is real).

## Decision

The Go 1.27 standard library is the **first choice**. An external dependency is
allowed only when stdlib is provably weaker. Final list of **3 dependencies**:

| Need | Package | Why not stdlib |
|---|---|---|
| Password hashing (Argon2id) | `golang.org/x/crypto` v0.55.0 | `crypto/argon2` is not in stdlib; the alternative, PBKDF2 (`crypto/pbkdf2`), is weaker per [OWASP 2024] |
| `rate.Limiter` for `/login` | `golang.org/x/time` v0.16.0 | No stdlib analog; writing it from scratch is ~150 lines with tests — not justified |
| SQLite without CGO | `modernc.org/sqlite` v1.60.1 | `mattn/go-sqlite3` requires CGO; that kills `CGO_ENABLED=0`. `ncruces/go-sqlite` exists but is less battle-tested |

## Alternatives we rejected

- **`chi`/`gin`/`echo`/`fiber`** — `net/http` 1.22+ covers pattern routing (`{id}`,
  `{$}`), middleware chain, everything we need.
- **`gorm`/`sqlx`/`ent`** — parameterized `?`-queries are simpler and more
  transparent. `ent` especially loves reflection, which hurts cold start.
- **`gorilla/sessions`** — Cookie sessions in DB are ~50 lines, readable and
  testable.
- **`gorilla/csrf`** — `http.NewCrossOriginProtection()` is in stdlib 1.25+.
- **`alexedwards/scs`** — see sessions.
- **JWT for sessions** — see [ADR-0002](0002-argon2id-and-cookie-sessions.md).

## Consequences

**Pros:**

- Binary size: ~13 MB (without deps it would be ~10 MB, but we need Argon2 anyway).
- Upgrade cycle is tied to Go releases: Go 1.27 brought `uuid` into stdlib and
  `http.CrossOriginProtection` — without touching deps.
- CVE audit — 3 deps, all hand-checked via `govulncheck`.
- Onboarding — a new developer knows `net/http`; they do not have to learn a router.

**Cons:**

- Everything is "ours" — password reset, OAuth, email verification. This is a
  conscious compromise for the template; see [architecture.md](../architecture.md)
  (section "What is NOT in the template").
- When upgrading Go 1.27 → 1.28+ nothing should break; but there is no
  guarantee the Go team keeps `uuid` / `CrossOriginProtection` forever.
- If SSE/WebSocket becomes a real need, we write it ourselves (no `gobwas/ws`).
  When the recipe lands, we revisit.

## When to revisit

- If Go 1.30+ deprecates something we rely on (for example, marks
  `CrossOriginProtection` as deprecated) — revisit.
- If the project grows into a real multi-service architecture, a 4th dependency
  may appear, but **only** for a concrete task (e.g. a NATS client).

## Decision dependencies

- [ADR-0002](0002-argon2id-and-cookie-sessions.md) — why cookie sessions in DB, not JWT.
- [ADR-0003](0003-no-orm-no-router.md) — why no ORM and no router.
