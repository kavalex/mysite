# Recipe: replace SQLite with Postgres

**Status:** placeholder.

When SQLite stops keeping up (> 100 RPS sustained, JOINs over 5+ tables,
> 100 GB), migrate to Postgres. Points of change:

1. **Driver.** `_ "modernc.org/sqlite"` → `_ "github.com/jackc/pgx/v5/stdlib"`
   (or native `pgx`). This is a **4th dependency**; see [../adr/0001-stdlib-first.md](../adr/0001-stdlib-first.md).
2. **`store.Open`** — different DSN format, different pragmas; JSON functions
   differ (`json_extract` in SQLite vs `jsonb` in PG).
3. **Migrations** — Alembic / Atlas / sqitch / migrate-style. The template uses
   embed.FS; for PG you either rewrite `store.Migrate` or pick a tool.
4. **Pool sizing.** `SetMaxOpenConns(1)` is a SQLite-specific constraint. For PG,
   set `SetMaxOpenConns(20)` or `2 * runtime.NumCPU() + 1`.
5. **Types.** SQLite does not distinguish INTEGER and BIGINT; PG does. You may
   need to revisit migrations (timestamps in int64 milliseconds, not Unix
   seconds).
6. **`pgcrypto` / UUID.** PG has a built-in UUID type; you could drop the
   stdlib `uuid` import — but we never made it a hard dependency anyway, it is
   already stdlib.

The recipe will appear the first time someone runs through this migration.
