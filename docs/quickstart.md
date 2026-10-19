# Quickstart

Launch a dev server in 5 minutes. No Docker, no Node.js, no CGO.

## 1. Install Go 1.27+

```bash
go version    # expected: go1.27.x or newer
```

If Go is not installed, the easiest path is the official distribution:
- macOS: `brew install go`
- Linux: see https://go.dev/dl/

## 2. Clone and build

```bash
git clone <repo-url> mysite
cd mysite
make deps      # ≡ go mod download && go mod tidy
```

## 3. Configuration

```bash
cp .env.example .env
```

Defaults are safe for dev. You don't need to change anything for a local launch.

## 4. Run

```bash
make run       # ≡ make build && ./dist/mysite
```

The server listens on `http://localhost:8080`.

## 5. Verify

```bash
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/healthz   # 200
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/readyz    # 200
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/          # 200
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/about     # 200
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/account   # 302 (→ /login, no session)
curl -s http://localhost:8080/sitemap.xml | head -3
curl -s http://localhost:8080/robots.txt
```

## 6. What's next?

- **Add a page** → [recipes/new-page.md](recipes/new-page.md).
- **Add a DB entity** → [recipes/new-entity.md](recipes/new-entity.md).
- **Understand why everything is the way it is** → [architecture.md](architecture.md).
- **Deploy to production** → [deploy.md](deploy.md).

## Known dev-environment caveats

- **favicon.ico** — empty stub (0 bytes). Browsers will get a 200 with an empty body on `/favicon.ico`. This does not block development.
- **First user.** The template has no registration form — by design. To create a test user, use `cmd/genhash/main.go` + `sqlite3 data.db` (see [CONTRIBUTING.md §dev-tools](../CONTRIBUTING.md)).

## Common problems

| Symptom | Cause | Solution |
|---|---|---|
| `no such table: users` | `Migrate()` was not called | Already called in `main.go`; check that `data.db` exists |
| `database is locked` | multiple writer connections | Already handled by `SetMaxOpenConns(1)` in `store.Open` |
| 403 on POST | `http.CrossOriginProtection` blocked a cross-site request | Expected for curl without `Sec-Fetch-Site: same-origin` |
| inline-script does not work | CSP-header nonce and `<script>` nonce out of sync | Should never happen — see `anti-patterns.md` |
| `no matching files` on `//go:embed` | missing `web/templates/` or `web/static/` | Should never happen — structure is scaffolded |
