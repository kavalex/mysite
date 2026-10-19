package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("store: not found")

type User struct {
	ID           int64
	Email        string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (s *Store) CreateUser(ctx context.Context, email, username, passwordHash string) (int64, error) {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, username, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		email, username, passwordHash, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, username, password_hash, created_at, updated_at
		 FROM users WHERE email = ?`, email))
}

func (s *Store) UserByID(ctx context.Context, id int64) (*User, error) {
	return s.scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, email, username, password_hash, created_at, updated_at
		 FROM users WHERE id = ?`, id))
}

func (s *Store) scanUser(row *sql.Row) (*User, error) {
	var u User
	var created, updated int64
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.PasswordHash, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt = time.Unix(created, 0)
	u.UpdatedAt = time.Unix(updated, 0)
	return &u, nil
}
