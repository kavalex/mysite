# Contributing

Thanks for your interest. This template is intentionally minimal — so any change must
**fix a boundary** (documented in `docs/adr/` and `docs/architecture.md`), **fix a bug**,
or **add a recipe** (for a task you have hit more than once).

## Workflow

1. Open an issue or start with a draft in Discussions for non-trivial changes (new middleware,
   new dependency, structural changes).
2. Fork, branch, commit. Small commits preferred.
3. Make sure CI passes (if configured):
   - `gofmt -l .` is empty;
   - `go vet ./...` is empty;
   - `go test -race ./...` PASS;
   - `golangci-lint run --timeout=5m` 0 issues;
   - `govulncheck ./...` 0 vulnerabilities;
   - `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/mysite .` ok.
4. Open a PR with the "docs/ updated" checkbox if the change affects public behavior.

## What to discuss before opening a PR

| Change | Where to discuss | When a new ADR is required |
|---|---|---|
| New dependency | issue | ✓ if there are stdlib alternatives |
| Change to a public type's API (`server.New`, `handlers.New`, `store.Open`) | issue | ✓ |
| Change to middleware chain | issue + diagram | ✓ if order changes |
| Change to CSP / HSTS / cookies | issue | ✓ this is a security decision |
| Refactor of an internal package | freely | — |
| New page / handler / recipe | PR without discussion | — |
| New test | PR without discussion | — |
| New recipe in `docs/recipes/` | PR without discussion | — |

## Code style

- `gofmt` (no arguments). `goimports` with local prefix `mysite`.
- Public type names are `MixedCaps`, no `I`/`C` prefix.
- Errors are wrapped: `fmt.Errorf("create user: %w", err)`.
- Logs via `slog` through middleware; handlers only use `slog.ErrorContext`.
- Templates use `html/template`; `template.HTML(...)` is forbidden without explicit justification in the PR.

## Security

**Don't open public issues for vulnerabilities.** Use [SECURITY.md](SECURITY.md).

## Dev tools

These helpers live in the repo and are not part of the deployable binary:

- `cmd/genhash/main.go` — generate an Argon2id hash for a fresh user.
  Usage:
  ```bash
  HASH=$(go run ./cmd/genhash 'correct-horse-staple-42')
  sqlite3 data.db "INSERT INTO users (email, username, password_hash, created_at, updated_at) VALUES ('alice@example.com', 'alice', '$HASH', strftime('%s','now'), strftime('%s','now'));"
  ```

After changes to `auth.HashPassword` regenerate any seeded users.

## License

By submitting a PR, you agree that your contribution is licensed under [MIT](LICENSE).
