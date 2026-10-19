package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kavalex/mysite/internal/auth"
	"github.com/kavalex/mysite/internal/config"
	"github.com/kavalex/mysite/internal/handlers"
	"github.com/kavalex/mysite/internal/store"
)

func setupTestDB(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(&config.Config{DatabaseURL: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLoginSubmit(t *testing.T) {
	s := setupTestDB(t)

	hash, err := auth.HashPassword("correct-horse-staple-42")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser(context.Background(), "test@example.com", "testuser", hash); err != nil {
		t.Fatal(err)
	}

	h := handlers.New(&config.Config{Env: "dev"}, s)

	cases := []struct {
		name     string
		email    string
		password string
		want     int
	}{
		{"valid", "test@example.com", "correct-horse-staple-42", http.StatusFound},
		{"wrong password", "test@example.com", "wrong", http.StatusUnauthorized},
		{"nonexistent", "nobody@example.com", "correct-horse-staple-42", http.StatusUnauthorized},
		{"empty email", "", "correct-horse-staple-42", http.StatusUnauthorized},
		{"sql injection", "test@example.com' OR 1=1 --", "x", http.StatusUnauthorized},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			form := url.Values{"email": {c.email}, "password": {c.password}}
			req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			h.LoginSubmit(w, req)

			if w.Code != c.want {
				t.Fatalf("got %d, want %d (body: %s)", w.Code, c.want, w.Body.String())
			}
		})
	}
}
