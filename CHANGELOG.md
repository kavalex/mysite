# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `docs/` — Template Guide (architecture, recipes, reference, ADRs, anti-patterns, deploy, quickstart).
- `CONTRIBUTING.md`, `CHANGELOG.md`.

### Changed
- `README.md` redesigned as a project map; operational details moved to `docs/quickstart.md` and `docs/deploy.md`.

## [0.1.0] — 2026-10-19

### Added
- Base template: auth (argon2id + cookie sessions), CSRF (`http.NewCrossOriginProtection`), CSP nonce, HSTS, rate-limit on `/login`.
- Healthchecks (`/healthz`, `/readyz`), `X-Request-Id`, structured slog, SIGHUP-reload.
- SQLite + migrations (`modernc.org/sqlite`, pure-Go).
- Pages: home, about, login, account. robots.txt, sitemap.xml.
- Tests: argon2id (including DoS protection), login (including SQL injection), rate-limit (`testing/synctest`).
- `Makefile`, `.golangci.yml` (v2), `.env.example`.

[Unreleased]: https://example.com/mysite/compare/v0.1.0...HEAD
[0.1.0]: https://example.com/mysite/releases/tag/v0.1.0
