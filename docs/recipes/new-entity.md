# Recipe: add a DB entity

Goal: add `posts` (blog posts) with CRUD for authenticated users and a public
feed. The example covers: migration, store methods, handler, templates,
protection.

## 1. Migration

Create `internal/store/migrations/0002_posts.sql`:

```sql
CREATE TABLE IF NOT EXISTS posts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    slug        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    author_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    published   INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_posts_slug      ON posts(slug);
CREATE INDEX IF NOT EXISTS idx_posts_author    ON posts(author_id);
CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(published, created_at);
```

The migration is applied automatically on the next server start. SQLite specifics:
`INTEGER PRIMARY KEY` instead of `SERIAL`; timestamps as `INTEGER` (Unix).

## 2. Model in Store

Create `internal/store/posts.go`:

```go
package store

import (
    "context"
    "database/sql"
    "errors"
    "time"
)

type Post struct {
    ID        int64
    Slug      string
    Title     string
    Body      string
    AuthorID  int64
    Published bool
    CreatedAt time.Time
    UpdatedAt time.Time
}

func (s *Store) CreatePost(ctx context.Context, slug, title, body string, authorID int64) (int64, error) {
    now := time.Now().Unix()
    res, err := s.db.ExecContext(ctx,
        `INSERT INTO posts (slug, title, body, author_id, published, created_at, updated_at)
         VALUES (?, ?, ?, ?, 0, ?, ?)`,
        slug, title, body, authorID, now, now)
    if err != nil {
        return 0, err
    }
    return res.LastInsertId()
}

func (s *Store) PostBySlug(ctx context.Context, slug string) (*Post, error) {
    var p Post
    var published int
    var created, updated int64
    err := s.db.QueryRowContext(ctx,
        `SELECT id, slug, title, body, author_id, published, created_at, updated_at
         FROM posts WHERE slug = ?`, slug).
        Scan(&p.ID, &p.Slug, &p.Title, &p.Body, &p.AuthorID, &published, &created, &updated)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }
    p.Published = published != 0
    p.CreatedAt = time.Unix(created, 0)
    p.UpdatedAt = time.Unix(updated, 0)
    return &p, nil
}

func (s *Store) PostsList(ctx context.Context, limit int) ([]*Post, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, slug, title, body, author_id, published, created_at, updated_at
         FROM posts WHERE published = 1 ORDER BY created_at DESC LIMIT ?`, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var posts []*Post
    for rows.Next() {
        var p Post
        var published int
        var created, updated int64
        if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Body, &p.AuthorID, &published, &created, &updated); err != nil {
            return nil, err
        }
        p.Published = published != 0
        p.CreatedAt = time.Unix(created, 0)
        p.UpdatedAt = time.Unix(updated, 0)
        posts = append(posts, &p)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }
    return posts, nil
}
```

Parameterized queries throughout. No string concatenation.

## 3. Handlers

In `internal/handlers/handlers.go` or in a new file (when handlers grow):

```go
// The constructor already exposes *store.Store.

func (h *Handler) PostsList(w http.ResponseWriter, r *http.Request) {
    posts, err := h.store.PostsList(r.Context(), 20)
    if err != nil {
        h.internalError(w, r, err)
        return
    }
    pd := h.pageData(r)
    pd.Title = "Posts"
    pd.Data = posts
    h.render(w, r, "posts-list", pd)
}

func (h *Handler) PostView(w http.ResponseWriter, r *http.Request) {
    slug := r.PathValue("slug")
    p, err := h.store.PostBySlug(r.Context(), slug)
    if errors.Is(err, store.ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        h.internalError(w, r, err)
        return
    }
    pd := h.pageData(r)
    pd.Title = p.Title
    pd.Data = p
    h.render(w, r, "post-view", pd)
}

// Authenticated only:
func (h *Handler) PostCreate(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    slug := strings.TrimSpace(r.PostFormValue("slug"))
    title := strings.TrimSpace(r.PostFormValue("title"))
    body := r.PostFormValue("body")

    if slug == "" || title == "" || body == "" {
        http.Error(w, "slug, title, body required", http.StatusBadRequest)
        return
    }

    uid, ok := auth.UserIDFrom(r.Context())
    if !ok {
        h.internalError(w, r, errors.New("no user in context"))
        return
    }

    _, err := h.store.CreatePost(r.Context(), slug, title, body, uid)
    if err != nil {
        h.internalError(w, r, err)
        return
    }
    http.Redirect(w, r, "/posts/"+slug, http.StatusFound)
}
```

## 4. Routes

In `internal/server/server.go`:

```go
requireLogin := auth.RequireLogin(s.DB(), secure)

// Public
mux.HandleFunc("GET /posts", h.PostsList)
mux.HandleFunc("GET /posts/{slug}", h.PostView)

// Authenticated only
mux.Handle("POST /posts", requireLogin(http.HandlerFunc(h.PostCreate)))
```

## 5. Tests

Either `internal/store/posts_test.go` (direct Store testing) or
`internal/handlers/posts_test.go` (HTTP-level, like `login_test.go`):

```go
func TestPostsList(t *testing.T) {
    s := setupTestDB(t)
    s.CreateUser(context.Background(), "alice@example.com", "alice", "<hash>")
    s.CreatePost(context.Background(), "hello", "Hello", "world body", 1)

    posts, err := s.PostsList(context.Background(), 10)
    if err != nil { t.Fatal(err) }
    if len(posts) != 1 { t.Fatalf("got %d posts", len(posts)) }
}
```

## 6. Templates

`web/templates/pages/posts-list.html`:

```html
{{ define "title" }}Posts — MySite{{ end }}

{{ define "content" }}
<h1>Posts</h1>
<ul>
  {{ range .Data }}
    <li><a href="/posts/{{ .Slug }}">{{ .Title }}</a></li>
  {{ else }}
    <li>No posts yet.</li>
  {{ end }}
</ul>
{{ end }}
```

`web/templates/pages/post-view.html`:

```html
{{ define "title" }}{{ .Data.Title }} — MySite{{ end }}

{{ define "content" }}
<article>
  <h1>{{ .Data.Title }}</h1>
  <p>{{ .Data.Body }}</p>
</article>
{{ end }}
```

## 7. Sitemap

Extend `internal/handlers/sitemap.go` if you want published posts in the
sitemap:

```go
// When articles/posts appear:
// rows, _ := h.store.DB().QueryContext(r.Context(),
//     `SELECT slug FROM posts WHERE published = 1`)
// defer rows.Close()
// for rows.Next() { ... }
```

(Already commented in the template.)

## Security checklist

- [ ] **All DB queries use `?`-placeholders.** No concatenation.
- [ ] **`sql.ErrNoRows` is handled explicitly** via `errors.Is(err, sql.ErrNoRows) → ErrNotFound`.
- [ ] **Slug is validated for allowed characters** before `CreatePost` (e.g. `^[a-z0-9-]+$`).
      Add `Slug()` to [internal/validate/validate.go](../../internal/validate/validate.go).
- [ ] **`published` is `bool` / `INTEGER NOT NULL`**, not `TEXT` — fewer surprises.
- [ ] **`ON DELETE CASCADE`** for `author_id` — otherwise deleting a user is blocked by FK violation.
- [ ] **In the template `{{ }}` without `template.HTML`** — post bodies are escaped.
      If the body is Markdown, add `goldmark` (or similar) and sanitize its output.
- [ ] **Index on `slug`** — for `WHERE slug = ?` (in the migration).
- [ ] **Index on `(published, created_at)`** — for `WHERE published = 1 ORDER BY created_at DESC` (in the migration).

## What NOT to do

- **Do not use `text/template`.** Different package, no XSS protection.
- **Do not make `posts.body` a JSON field** — it is text, not structure.
      If you need nesting, use a separate table.
- **Do not cascade `posts` on `users` without thinking.** Possibly you need
      soft-delete on users (`deleted_at`) so posts survive.
- **Do not return `*Post` on error** — `*sql.Row` after an error has undefined fields.

## See also

- [architecture.md](../architecture.md) (section "What is flexible") — exactly where to edit
- [reference.md](../reference.md) (section "File structure") — where to drop the migration
- [anti-patterns.md](../anti-patterns.md) (section "SQLite / tests") — in-memory SQLite and `ErrNoRows`
- [swap-db.md](swap-db.md) — what changes when migrating from SQLite to Postgres
