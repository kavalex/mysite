package store

import (
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/kavalex/mysite/internal/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	db *sql.DB
}

func (s *Store) DB() *sql.DB { return s.db }

func Open(cfg *config.Config) (*Store, error) {
	dsn := cfg.DatabaseURL
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}

	pragmas := "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	// WAL is not supported for in-memory databases.
	if !strings.Contains(dsn, ":memory:") {
		pragmas = "_pragma=journal_mode(WAL)&" + pragmas
	}
	dsn += sep + pragmas

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	// SQLite: single writer. For in-memory this is critical — otherwise
	// each connection gets its own DB and migrations "do not see" test queries.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    TEXT PRIMARY KEY,
		applied_at INTEGER NOT NULL
	)`); err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		version := e.Name()

		var existing string
		err := s.db.QueryRow("SELECT version FROM schema_migrations WHERE version = ?", version).Scan(&existing)
		if err == nil {
			continue // already applied
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		sqlBytes, err := fs.ReadFile(migrationsFS, "migrations/"+version)
		if err != nil {
			return err
		}

		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
			version, time.Now().Unix()); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
