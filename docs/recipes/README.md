# Recipes

Step-by-step how-tos for extending the template. One recipe — one task.
Recipes reference [architecture.md](../architecture.md) for justifications
and [anti-patterns.md](../anti-patterns.md) for pitfalls.

## Where to start

| Task | Recipe | Difficulty |
|---|---|---|
| Add an HTML page | [new-page.md](new-page.md) | trivial |
| Add a DB entity | [new-entity.md](new-entity.md) | typical |
| Add a JSON API | [add-api.md](add-api.md) | typical |
| Replace SQLite with Postgres | [swap-db.md](swap-db.md) | advanced |
| Add OAuth (Google/GitHub) | [add-oauth.md](add-oauth.md) | advanced |
| Add SSE / WebSocket | [add-realtime.md](add-realtime.md) | advanced |
| Background jobs | [add-job.md](add-job.md) | advanced |

Recipes marked with the placeholder note will be filled in when somebody first
hits the task — they will write the real recipe in a PR. Until then —
[architecture.md](../architecture.md) (section "What is NOT in the template")
records what is out of scope for MVP.

## Recipe structure

Every file in this directory:

1. **Verb in the title** ("Add", "Replace", "Connect").
2. **Minimal "why"** — one link at the end to `architecture.md`.
3. **Numbered steps** with minimal listings (copy-pasteable).
4. **Security checklist** at the end — typical pitfalls specific to the recipe.
5. **See also** — links to architecture, anti-patterns, related recipes.

Recipes **do not explain the API or re-describe reference** — that lives in
[reference.md](../reference.md).
