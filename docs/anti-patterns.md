# Anti-patterns

Each item is a mistake that **keeps coming back**. For each: why it breaks and
how to do it right. See also [architecture.md](architecture.md) (section "Anti-patterns").

## Security

### 1. `template.HTML(...)` on user input

❌

```go
render.Page(w, "home", &render.PageData{
    Data: template.HTML(req.FormValue("name")),  // XSS!
})
```

✅

```go
render.Page(w, "home", &render.PageData{
    Data: req.FormValue("name"),  // html/template escapes it
})
```

`template.HTML` disables escaping **entirely**. One call with unvalidated data is
a stable XSS. If you really need HTML (Markdown, WYSIWYG), bring in
`github.com/microcosm-cc/bluemonday` and sanitize **before** wrapping in
`template.HTML`. See [architecture.md](architecture.md).

### 2. Hand-rolled CSRF middleware

❌ Any CSRF written inside the project has ~90% probability of a bypass.

✅ `http.NewCrossOriginProtection()` from `net/http` (Go 1.25+). One call
in [internal/server/server.go](../internal/server/server.go). Protection via
`Sec-Fetch-Site` and `Origin`, not via tokens.

### 3. Long tokens in URLs (?token=...)

❌ They end up in access-log files, in Referer, in browser history.

✅ Short-lived tokens in cookies with `SameSite=Lax` + `Secure` (prod), or a
magic link in URL **with a hash, single-use and short TTL**.

### 4. Plaintext passwords in logs

❌

```go
slog.Info("login attempt", "email", email, "password", password)
```

✅ Log only `email`, `success bool`, `request_id`. Do not log whole request
bodies — they may contain anything.

### 5. SQL string concatenation

❌

```go
db.QueryRow("SELECT * FROM users WHERE email = '" + email + "'")
```

✅ Parameterized `?`-placeholders:

```go
db.QueryRowContext(ctx, "SELECT * FROM users WHERE email = ?", email)
```

See [OWASP Top 10 A03: Injection](https://owasp.org/Top10/A03_2021-Injection/).
In this template's code we use parameterized queries throughout — no concatenation.

### 6. Re-reading a request `Body` more than once

❌ `Body io.Reader` is single-use. After reading, `EOF`. Reusing it = empty body,
silent data loss.

✅ If you must read twice, use `io.TeeReader` to drain into a buffer first, then
make new readers from that buffer. Not needed in this template; remember it when
you extend it (for example, multipart parsing with validation).

## SQLite / tests

### 7. `:memory:` SQLite without `SetMaxOpenConns(1)`

❌ Each connection gets its own in-memory DB. Migrations apply to one, the test
queries hit another. Tests pass, production breaks.

✅ Already done in [`internal/store/store.go`](../internal/store/store.go).

### 8. Ignoring `sql.ErrNoRows`

❌

```go
var u User
err := db.QueryRow("SELECT ...").Scan(&u.ID, &u.Email)
if err != nil {
    return u, err  // returning a partially populated User
}
```

✅

```go
var u User
err := db.QueryRow("SELECT ...").Scan(&u.ID, &u.Email)
if errors.Is(err, sql.ErrNoRows) {
    return nil, ErrNotFound
}
if err != nil {
    return nil, err
}
```

`*sql.Row` after an error has undefined fields; do not return it.

## Templates

### 9. `{{ define "content" }}` without `base.Clone()`

❌ All pages show the same `{{ define "content" }}` — the one loaded last.

✅ Already done in [`internal/render/render.go:init`](../internal/render/render.go).
When you add a new page just drop the file in `web/templates/pages/`; `render.init`
picks it up automatically.

### 10. Out-of-sync nonce in CSP header and `<script>`

❌ The nonce in `<script nonce="abc">` differs from `nonce-abc` in
`Content-Security-Policy: ...` — CSP blocks the inline script.

✅ Both are generated in one middleware [`SecurityHeaders`](../internal/middleware/headers.go)
**before** calling `next.ServeHTTP`. The handler reads the nonce via
`middleware.CSPNonceFrom(r.Context())`.

If you edit CSP — do not move nonce generation into the handler.

## Architecture

### 11. Replacing `net/http` with chi/echo/fiber

❌ Premature abstraction. `http.ServeMux` in Go 1.22+ covers pattern routing
(`/users/{id}`), middleware, everything you need.

✅ Stay on `net/http`. If you **really** miss something — issue + ADR.

### 12. Adding an ORM

❌ `gorm`, `sqlx`, `ent` — an extra layer, loss of SQL control, new attack
surface (`ent` in particular loves reflection).

✅ `database/sql` + hand-written parameterized queries. Already done.

### 13. Adding `github.com/google/uuid`

❌ In Go 1.27 `import "uuid"` is already in the stdlib.

✅

```go
import "uuid"
// ...
id := uuid.New().String()
```

### 14. Forgetting to call `store.Migrate` in `main.go`

❌ `Migrate()` runs the SQL from `embed.FS`; without it there are no tables.

✅ Already called in `main.go` right after `store.Open`. If you split it into a
separate migration CLI — say so; for now it lives in one process.

### 15. Long middleware chains (> 10 layers)

❌ Every middleware is a potential panic point and an extra allocation.

✅ Six layers today; before adding a new one ask "can this go in an existing one?".

## Operations

### 16. Secrets in `.env` in the repo

❌ `.env` in git = leaked keys within 5 minutes of a push.

✅ Only `.env.example` (no secrets). Real secrets go in `EnvironmentFile` for
systemd, Vault, Doppler, or Kubernetes Secrets.

### 17. `ENV=dev` in production

❌ Text logs, no `frame-ancestors`, no `X-Frame-Options`.

✅ `ENV=prod` (see the table in [reference.md](reference.md)).

### 18. Forgetting `ReadHeaderTimeout`

❌ Slowloris attack: client holds the TCP connection and sends headers slowly.

✅ `ReadHeaderTimeout` is set in [internal/server/server.go](../internal/server/server.go).
Default 5 seconds.

### 19. Docker without Kubernetes

❌ Extra abstraction layer, extra attack surface (the Docker daemon is a root process).

✅ One static binary + systemd. If you run k8s — that is a different conversation.

### 20. `make build` without `-trimpath -ldflags="-s -w"`

❌ Binary contains developer-local paths, symbol table, DWARF — larger, points
back to your laptop as `localhost`.

✅ `Makefile` already does this. Don't change the flags without good reason.
