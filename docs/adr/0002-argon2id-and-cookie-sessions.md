# ADR-0002: Argon2id and cookie sessions in DB

**Date:** 2026-10-14.
**Status:** accepted.

## Context

Two interrelated decisions:

1. **How do we store passwords?** Hashed (with salt), never plaintext.
2. **How do we authenticate requests?** Issue a token to the client and
   validate it on the server.

### Password hash: alternatives

| Option | Pros | Cons | Verdict |
|---|---|---|---|
| `bcrypt` (`golang.org/x/crypto/bcrypt`) | in stdlib-ext, proven | slower than Argon2id, worse GPU resistance | ✗ |
| `scrypt` (`golang.org/x/crypto/scrypt`) | in stdlib-ext | less studied, harder to tune | ✗ |
| `PBKDF2` (`crypto/pbkdf2`) | in stdlib | OWASP 2024 considers it weaker than Argon2id | ✗ |
| **Argon2id** (`crypto/argon2`) | OWASP 2024 recommended, GPU-resistant | not in stdlib — needs `golang.org/x/crypto` | ✓ |
| `MD5` / `SHA-256` plain | fast | absolutely unacceptable | ✗ |

### Sessions: alternatives

| Option | Logout | Invalidation by user_id | Server-side state | Transparency |
|---|:-:|:-:|:-:|---|
| **Cookie session in DB** | ✓ | ✓ | ✓ | ✓ |
| JWT (signed) | ✗ | ✗ | ✗ | ✓ |
| JWT + revocation list | ~ | ~ | ✓ | ✗ |
| Signed cookie (stateless) | ✗ | ✗ | ✗ | ✓ |

JWTs cannot be invalidated server-side in principle: the token stays valid
until expiry regardless of what the server thinks. "Logout" with JWT is a
request to the client to drop the token; the server accepts it until the
last second of validity.

A cookie session with the ID in DB: the server alone decides whether the
session is valid, and can kill it at any moment (logout, password change,
compromise).

## Decision

### Password hash — Argon2id with OWASP 2024 parameters

- `m=64 MiB` (Memory)
- `t=3` (Iterations)
- `p=4` (Parallelism)
- Salt: 16 random bytes via `crypto/rand`.
- Key: 32 bytes.
- Encoded format: `$argon2id$v=19$m=65536,t=3,p=4$<salt-b64>$<hash-b64>`.

### DoS protection on validation

If an attacker controls a hash (for example via SQLi in profile editing),
they could submit a hash with `m=1GB`. Argon2 would eat all memory and take
the service down.

The mitigation is in [`VerifyPassword`](../../internal/auth/password.go):

```go
const (
    maxMemory      = 256 * 1024 // 256 MiB
    maxIterations  = 10
    maxParallelism = 16
    maxKeyLength   = 64
)

// Validate parameters BEFORE running Argon2.
if p.Memory == 0 || p.Memory > maxMemory ||
    p.Iterations == 0 || p.Iterations > maxIterations ||
    p.Parallelism == 0 || p.Parallelism > maxParallelism {
    return false, errors.New("argon2 params out of range")
}
```

The test [`TestVerifyRejectsHugeParams`](../../internal/auth/password_test.go)
pins the contract: feeding a hash with `m=1048576 KiB` (= 1 GiB) returns an
error **before** the KDF runs.

### Session cookie — UUID in DB

- Session ID: `uuid.New().String()` (122 bits of entropy — 2^122 ≈ 5.3e36).
- Stored in the `sessions` table (see migration `0001_init.sql`).
- Cookie: `Name=session`, `Path=/`, `HttpOnly`, `SameSite=Lax`, `Secure` (only
  with TLS / in prod).
- TTL: 24 hours. On expiry the row is filtered out by `ValidateSession`.
- Logout: `DELETE FROM sessions WHERE id = ?`.
- Password change: `DELETE FROM sessions WHERE user_id = ?` (invalidate all
  sessions).

### Timing equalizer

For a login attempt with a non-existent email, run `VerifyPassword` on a
**dummy hash**. This takes about the same time as a real check, so a
timing-based enumeration attack via the login form becomes useless.

Implemented in [`internal/handlers/handlers.go` LoginSubmit](../../internal/handlers/handlers.go):
when `errors.Is(err, store.ErrNotFound)` — `_ = auth.VerifyPassword(password, h.dummyHash)`.

## Alternatives we rejected

### Hash

- **bcrypt** — OWASP prefers Argon2id, and `golang.org/x/crypto` is needed
  anyway (whether for bcrypt or Argon2); we picked the recommended one.
- **Custom SHA-256 + HMAC scheme** — that is not password hashing; it breaks
  fast. Don't.
- **Peppering (extra server-side secret)** — increases attack surface (a
  secret in code is also a secret you have to rotate). OWASP allows it but
  does not require it.

### Sessions

- **JWT** — cannot be invalidated.
- **JWT + revocation list** — loses JWT's state-less advantage while adding a DB.
- **Signed cookie** (`HMAC(payload, secret)`) — same problem as JWT, no
  invalidation.
- **Macaroon / cookie with delegated scopes** — overkill for the template.

## Consequences

### Pros

- Argon2id — modern protection, vetted parameters.
- Cookie sessions give explicit logout and session rotation on password change — no UX surprises.
- The timing equalizer makes enumeration through the login form meaningless.

### Cons

- Argon2id per login — ~50–200 ms CPU. At 10 RPS sustained this is fine; at
  1000 RPS it adds up. A verified-login cache will be needed eventually (see below).
- The DB grows by ~50 bytes per session. Auto-cleanup on expiry handles it.

### When to revisit

- If sustained traffic exceeds 100 RPS — add an in-memory verified-session
  cache (`sync.Map` or `ristretto`) with TTL == sessionDuration.
- If you go multi-node — sessions are already in the DB, all good; you may
  need sticky sessions or a shared cache.

## Decision dependencies

- [ADR-0001](0001-stdlib-first.md) — why `golang.org/x/crypto` is on the list.
- [ADR-0003](0003-no-orm-no-router.md) — why sessions live in SQLite, not Redis.
