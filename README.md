# MySite

> A Go web-application template: stdlib + 3 dependencies. Auth, security, deploy-ready.

**TL;DR**

```bash
cp .env.example .env
make build && ./dist/mysite
curl http://localhost:8080/healthz   # → 200
```

**What it is:** a minimal, secure, production-ready Go 1.27 web template.
Auth (argon2id + cookie sessions), CSRF, CSP, rate-limit, healthchecks, SIGHUP-reload,
SQLite migrations — built from the standard library and exactly three vetted dependencies.
A single static ~13 MB binary, no Docker / Node.js / CGO.

**What is inside:**

- **Stdlib-first.** `net/http` + `http.ServeMux`, `html/template`, `database/sql`, `crypto/rand`, `crypto/subtle`, `log/slog`, the `uuid` package, `http.NewCrossOriginProtection` — all from Go 1.27.
- **3 deps:** `golang.org/x/crypto` (argon2id), `golang.org/x/time` (`rate.Limiter`), `modernc.org/sqlite` (pure-Go SQLite).
- **Secure by default:** CSP with per-request nonce, HSTS, secure cookies (HttpOnly, SameSite=Lax), Argon2id with parameter validation.

**Status:** stable. Compatible with Go 1.27+. Verified: `gofmt`, `go vet`, `go test -race`, `golangci-lint v2`, `govulncheck`.

**Documentation — start here:**

| | |
|---|---|
| 🚀 [Quickstart](docs/quickstart.md) | launch a dev server in 5 minutes |
| 🏛 [Architecture](docs/architecture.md) | why these decisions; what must not change |
| 📚 [Recipes](docs/recipes/) | how to add a page, an entity, an API, OAuth, real-time… |
| 📖 [Reference](docs/reference.md) | env variables, routes, file structure |
| 🚢 [Deploy](docs/deploy.md) | reverse proxy, systemd, backups |
| ⚠️ [Anti-patterns](docs/anti-patterns.md) | what not to do, and why |
| 📐 [ADRs](docs/adr/) | records of key decisions |

**Security:** see [SECURITY.md](SECURITY.md) — where to report vulnerabilities (not as a public issue).

**License:** [MIT](LICENSE).
