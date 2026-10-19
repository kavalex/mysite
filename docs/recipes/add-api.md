# Recipe: add a JSON API

**Status:** placeholder. The recipe will be filled in when someone first adds a JSON endpoint.

In the meantime:

- Architectural decisions — [../architecture.md](../architecture.md). Sections
  "What is flexible" and "Request flow".
- [../anti-patterns.md](../anti-patterns.md) — also covers safety.
- An HTML template vs a JSON API is usually one page and two handlers
  (`h.Foo` renders HTML, `h.FooAPI` returns `application/json`); look at how
  `LoginSubmit` is built for an example.
- For CSRF on the API — `http.NewCrossOriginProtection()` in
  [../../internal/server/server.go](../../internal/server/server.go) protects
  every unsafe method (POST/PUT/DELETE included) the same way; JSON endpoints
  work the same (relies on `Origin`/`Sec-Fetch-Site`).

The real recipe appears within 1–2 PRs after the first such task.
