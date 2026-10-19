# Recipe: add an HTML page

Goal: add a `/pricing` page, accessible to everyone, no auth.

## 1. Template

Create `web/templates/pages/pricing.html`:

```html
{{ define "title" }}Pricing — MySite{{ end }}

{{ define "content" }}
<h1>Plans</h1>
<p>Free — $0. Paid — $100 per month.</p>
{{ end }}
```

The file name (without `.html`) becomes the page name for
`render.Page(w, "pricing", ...)`. `render.init` picks it up automatically on the
next start.

## 2. Handler

In `internal/handlers/handlers.go`:

```go
func (h *Handler) Pricing(w http.ResponseWriter, r *http.Request) {
    pd := h.pageData(r)
    pd.Title = "Pricing"
    h.render(w, r, "pricing", pd)
}
```

## 3. Route

In `internal/server/server.go`, in the public-routes group:

```go
mux.HandleFunc("GET /pricing", h.Pricing)
```

`http.ServeMux` (Go 1.22+) supports patterns (`{$}` for root, `{slug}` for
parameters). No third-party routers needed.

## 4. Smoke test

```bash
make run                          # in another terminal
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/pricing   # 200
curl -s http://localhost:8080/pricing | grep -E '<title>|<h1>'           # expected content
```

If you do not get 200, check the logs; usually it is a typo in the template name (`render.unknown page`).

## 5. Add to sitemap (if you want)

In `internal/handlers/sitemap.go`, extend `entries`:

```go
entries := []sitemapURL{
    {Loc: base + "/"},
    {Loc: base + "/about"},
    {Loc: base + "/pricing"},  // ← new
}
```

## Security checklist

- [ ] **All user input goes through `{{ }}`** — `html/template` escapes automatically.
- [ ] **No `template.HTML(...)`** without an explicit justified PR.
- [ ] **If the page has a form:** it sits behind `MaxBodySize` middleware and
      under `http.NewCrossOriginProtection()` (which is already true — middleware
      is hung on the whole mux).
- [ ] **CSP nonce** is only needed if you write an inline `<script>` or `<style>`.
      See [anti-patterns.md §10](../anti-patterns.md).
- [ ] **`robots.txt`** — if a page is public but not indexable (`/pricing/admin`)
      → add `Disallow: /pricing/admin` in `static/robots.txt`.

## What NOT to do

- **Do not put business logic in the template.** Anything computed (prices with
  discounts, user roles) — go in the handler, into `PageData.Data`.
- **Do not pull in a new JavaScript framework.** If you need more interactivity,
  look at htmx — it is 10 KB and does not break the template.

## See also

- [architecture.md](../architecture.md) (section "What is flexible") — why this is the right place to change
- [anti-patterns.md](../anti-patterns.md) (section "Templates") — what happens if you forget `base.Clone()`
- [new-entity.md](new-entity.md) — if the page references DB data
- [add-api.md](add-api.md) — if you want a JSON variant of the same page
