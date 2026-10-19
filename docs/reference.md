# Reference

Factual information about the application: env variables, routes, file structure.
For justifications — [architecture.md](architecture.md). For how to change things — [recipes/](recipes/).

## Environment variables

Full list with defaults — in [.env.example](../.env.example).

Canonical table (maintained by hand when `internal/config/config.go` changes):

| Name | Type | Default | SIGHUP-reload | Description |
|---|---|---|:-:|---|
| `ENV` | string | `dev` | ✓ | `dev` or `prod`. Affects `X-Frame-Options`, log format (text vs JSON). |
| `ADDR` | string | `:8080` | ✗ | listen address `host:port`. Restart required to change. |
| `DATABASE_URL` | string | `data.db` | ✗ | Path to SQLite, or `:memory:` for tests. Supports query-string pragmas (`?WAL=...`). |
| `BASE_URL` | string | `http://localhost:8080` | ✗ | Canonical origin for sitemap, canonical-link. **No trailing slash.** |
| `READ_TIMEOUT` | duration | `5s` | ✗ | for `http.Server.ReadTimeout`. |
| `WRITE_TIMEOUT` | duration | `10s` | ✗ | for `http.Server.WriteTimeout`. |
| `IDLE_TIMEOUT` | duration | `120s` | ✗ | for `http.Server.IdleTimeout`. |
| `HEADER_TIMEOUT` | duration | `5s` | ✗ | for `http.Server.ReadHeaderTimeout`. |
| `LOGIN_RATE_PER_MIN` | int | `30` | ✓ | Tokens per minute per IP on `/login`. |
| `LOGIN_BURST` | int | `5` | ✓ | How many requests can run back-to-back without waiting. |
| `TRUSTED_PROXIES` | CSV | _(empty)_ | ✗ | CIDR/IP of trusted reverse-proxy. If empty, `X-Forwarded-For` is ignored entirely. |
| `TLS_CERT_FILE` | string | _(empty)_ | ✗ | If both `TLS_*` are set, the server comes up over HTTPS. |
| `TLS_KEY_FILE` | string | _(empty)_ | ✗ | Symmetric with `TLS_CERT_FILE`. |

### SIGHUP behavior

SIGHUP via `kill -HUP <pid>` or `systemctl reload mysite`:

```
SIGHUP: reloading config
SIGHUP: applied  field=ENV
SIGHUP: applied  field=LOGIN_RATE_PER_MIN
SIGHUP: applied  field=LOGIN_BURST
SIGHUP: change requires restart  field=ADDR
```

`applied` — the field was picked up live. `change requires restart` — the
change was seen in `Diff(next)` but not applied; requires `systemctl restart`.

If `.env` does not exist, SIGHUP silently re-reads env from the current process
(env > file by the `LoadEnvFile` rule). On parse failure — `SIGHUP: config
invalid, keeping old`.

## Routes

| Method | Path | Handler | Guard | Description |
|---|---|---|:-:|---|
| `GET` | `/healthz` | inline | — | liveness, always 200 |
| `GET` | `/readyz` | inline | — | readiness, 200 if `db.Ping` ok, else 503 |
| `GET` | `/static/{path}` | `http.FileServer` | — | cache 1 hour (`Cache-Control: public, max-age=3600`) |
| `GET` | `/sitemap.xml` | `h.Sitemap` | — | application/xml, cache 1 hour |
| `GET` | `/robots.txt` | `h.Robots` | — | text/plain, cache 24 hours |
| `GET` | `/{$}` | `h.Home` | — | root only (`{$}`) |
| `GET` | `/about` | `h.About` | — | |
| `GET` | `/login` | `h.LoginForm` | — | login form |
| `POST` | `/login` | `h.LoginSubmit` | CSRF + RateLimit | see below |
| `POST` | `/logout` | `h.Logout` | CSRF | deletes session in DB, clears cookie, → 302 `/` |
| `GET` | `/account` | `h.Account` | RequireLogin | requires a valid cookie session |

### POST /login

- Fields: `email`, `password`. multipart/form-data or urlencoded.
- Validation: `validate.Email`, `validate.Password` (≥ 12 runes).
- Hash — Argon2id with parameters from DB (or from `auth.DefaultParams` when creating).
- Success: `auth.CreateSession` → `SetSessionCookie` → `302 /`.
- Failure (no user / wrong password / invalid input): `renderLoginError` → `401` with the login page and the message "Invalid email or password".
- Timing-equalized: for a non-existent user, Argon2id is still run on a dummy hash.
- Rate-limit: `loginRL.Middleware` — burst=5, rate=30/min (defaults); configurable via env, reloadable via SIGHUP.

## File structure

Annotated: what is here and when to touch it.

| Path | Purpose | When to edit |
|---|---|---|
| `main.go` | entry; SIGHUP; graceful shutdown | on lifecycle changes |
| `go.mod`, `go.sum` | dependencies | on adding/updating deps |
| `Makefile` | targets `build`, `test`, `vet`, `lint`, `run`, `build-all`, `clean`, `deps`, `modernize` | as needed |
| `.env.example` | env template | on adding an env variable |
| `.gitignore` | `/dist/`, `/data.db*`, `.env`, `cover.out` | rarely |
| `.golangci.yml` | `golangci-lint v2.x` config | on enabling a new linter |
| `LICENSE`, `CHANGELOG.md`, `CONTRIBUTING.md` | meta | on release / policy change |
| `SECURITY.md` | how to report vulnerabilities | on process changes |
| `README.md`, `docs/` | documentation | on API/behavior changes |
| `internal/config/` | env loading, `.env` file, config diff for SIGHUP | on adding env variables |
| `internal/logging/` | `slog` with dynamic level for SIGHUP-reload | on changing the observability stack |
| `internal/server/server.go` | middleware chain, mux, health-checks | on adding a route / middleware |
| `internal/handlers/handlers.go` | page handlers (Home, About, LoginForm, LoginSubmit, Logout, Account) | on adding a page |
| `internal/handlers/sitemap.go` | XML serialization for sitemap via `xml.Encoder` | on changing sitemap pages |
| `internal/handlers/robots.go` | serves `static/robots.txt` | rarely |
| `internal/middleware/recovery.go` | `recover()` → 500 + log | rarely |
| `internal/middleware/requestid.go` | `uuid.New()` + `X-Request-Id` | on changing the ID source |
| `internal/middleware/logging.go` | `responseRecorder` + slog | on changing log fields |
| `internal/middleware/headers.go` | CSP nonce + security headers | on changing policies |
| `internal/middleware/maxbody.go` | `http.MaxBytesReader(1<<20)` | on changing the limit |
| `internal/middleware/ratelimit.go` | IP-keyed `rate.Limiter` with trusted proxies | on changing the rate-limit algorithm |
| `internal/auth/password.go` | Argon2id with parameter validation | rarely (security-critical) |
| `internal/auth/session.go` | create/validate/delete session, `RequireLogin` middleware | on changing the auth flow |
| `internal/store/store.go` | SQLite open + `SetMaxOpenConns(1)` + migrations | on changing driver or schema |
| `internal/store/users.go` | `User`, `CreateUser`, `UserByEmail`, `UserByID` | on changing the user model |
| `internal/store/migrations/*.sql` | DDL | on schema changes |
| `internal/validate/` | regex validators `Email`, `Username`, `Password` | on changing rules |
| `internal/render/render.go` | `html/template` cache, `base.Clone()` for pages | rarely |
| `web/embed.go` | `//go:embed templates` + `static` | — |
| `web/templates/layouts/base.html` | `<head>`, `<body>`, `<script nonce>`, partial calls | on layout changes |
| `web/templates/partials/{header,footer}.html` | navigation, copyright | on redesign |
| `web/templates/pages/{home,about,login,account}.html` | `{{ define "title" }}`, `{{ define "content" }}` | on adding a page |
| `web/static/css/main.css` | minimal styles | on redesign |
| `web/static/robots.txt` | User-agent rules + Sitemap URL | on policy changes |
| `web/static/img/favicon.ico` | _stub, 0 bytes_ | later |
| `cmd/genhash/main.go` | dev utility: generates an Argon2id hash for bootstrapping a user | on changing `auth.HashPassword` |

## Dependencies

Direct (see [ADR-0001](adr/0001-stdlib-first.md)):

| Package | Version | Purpose |
|---|---|---|
| `golang.org/x/crypto` | v0.55.0 | Argon2id (`crypto/argon2`) |
| `golang.org/x/time` | v0.16.0 | `rate.Limiter` for `/login` |
| `modernc.org/sqlite` | v1.60.1 | SQLite without CGO |

Transitive indirects (from `modernc.org/sqlite`, not our choice) are listed
in `go.sum`; they only affect the binary size.

## Public API signatures

Full godoc — `go doc`. Key ones:

### `server.New`

```go
func New(cfg *config.Config, s *store.Store) (*http.Server, *Runtime, error)
```

Returns a `*http.Server` (ready for `ListenAndServe`) and `*Runtime`
(holds the rate-limiter; `Close()` releases the cleanup goroutine).

### `handlers.New`

```go
func New(cfg *config.Config, s *store.Store) *Handler
```

### `auth.RequireLogin`

```go
func RequireLogin(db *sql.DB, secure bool) func(http.Handler) http.Handler
```

Middleware: with no/invalid cookie — redirect to `/login`.

### `store.Open`

```go
func Open(cfg *config.Config) (*Store, error)
```

Opens SQLite at `cfg.DatabaseURL`, sets `SetMaxOpenConns(1)`.

### `store.Migrate`

```go
func (s *Store) Migrate() error
```

Applies all migrations from `internal/store/migrations/*.sql` in alphabetical order.
Tracks applied ones via `schema_migrations`.

### `auth.HashPassword` / `VerifyPassword`

```go
func HashPassword(password string) (string, error)
func VerifyPassword(password, encoded string) (bool, error)
```

Argon2id v=19. `VerifyPassword` validates parameters from the hash (DoS protection).

## Tests

Coverage (as of 0.1.0):

| File | Tests |
|---|---|
| `internal/auth/password_test.go` | `TestPasswordHashAndVerify`, `TestVerifyRejectsHugeParams` |
| `internal/handlers/login_test.go` | `TestLoginSubmit` (5 cases: valid, wrong_password, nonexistent, empty_email, sql_injection) |
| `internal/middleware/ratelimit_test.go` | `TestRateLimiterBurst` (via `testing/synctest`) |

Run: `go test -race ./...`. With coverage: `go test -race -coverprofile=cover.out ./...`.

## Runtime metrics

Sizes and timings — from real runs on Go 1.27.2, macOS arm64:

| Metric | Value |
|---|---|
| Binary size (`dist/mysite`) | 13,224,610 bytes (~13 MB) |
| Cold start (until `Listening: 8080`) | <10 ms |
| Alloc per request (`/healthz`) | <1 KB |
| Test duration (no coverage, no `-race`) | ~5 s |
| Test duration (`-race`) | ~6 s |
