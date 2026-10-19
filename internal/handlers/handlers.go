package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kavalex/mysite/internal/auth"
	"github.com/kavalex/mysite/internal/config"
	"github.com/kavalex/mysite/internal/middleware"
	"github.com/kavalex/mysite/internal/render"
	"github.com/kavalex/mysite/internal/store"
	"github.com/kavalex/mysite/internal/validate"
)

type Handler struct {
	cfg       *config.Config
	store     *store.Store
	dummyHash string // timing equalizer for non-existent emails
}

func New(cfg *config.Config, s *store.Store) *Handler {
	dummy, err := auth.HashPassword("dummy-timing-equalizer")
	if err != nil {
		panic(err)
	}
	return &Handler{cfg: cfg, store: s, dummyHash: dummy}
}

func (h *Handler) secure() bool {
	return h.cfg.TLS != nil || h.cfg.Env == "prod"
}

// pageData fills in PageData for the template.
// No CSRF field: protection sits at http.CrossOriginProtection (§6).
func (h *Handler) pageData(r *http.Request) *render.PageData {
	pd := &render.PageData{
		Nonce:     middleware.CSPNonceFrom(r.Context()),
		Canonical: h.cfg.BaseURL + r.URL.Path,
	}
	if uid, ok := auth.UserIDFrom(r.Context()); ok {
		if u, err := h.store.UserByID(r.Context(), uid); err == nil {
			pd.User = &render.UserData{ID: u.ID, Username: u.Username, Email: u.Email}
		}
	}
	return pd
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	pd := h.pageData(r)
	pd.Title = "Home"
	h.render(w, r, "home", pd)
}

func (h *Handler) About(w http.ResponseWriter, r *http.Request) {
	pd := h.pageData(r)
	pd.Title = "About"
	h.render(w, r, "about", pd)
}

func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	pd := h.pageData(r)
	pd.Title = "Sign in"
	h.render(w, r, "login", pd)
}

func (h *Handler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.PostFormValue("email"))
	password := r.PostFormValue("password")

	if validate.Email(email) != nil || validate.Password(password) != nil {
		h.renderLoginError(w, r, "Invalid email or password")
		return
	}

	u, err := h.store.UserByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		// Equalize timing: Argon2 on the dummy hash takes about the same
		// time as on a real one — do not leak email existence.
		_, _ = auth.VerifyPassword(password, h.dummyHash)
		h.renderLoginError(w, r, "Invalid email or password")
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}

	ok, err := auth.VerifyPassword(password, u.PasswordHash)
	if err != nil || !ok {
		h.renderLoginError(w, r, "Invalid email or password")
		return
	}

	sid, err := auth.CreateSession(h.store.DB(), u.ID)
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	auth.SetSessionCookie(w, sid, h.secure())
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.SessionCookieName); err == nil {
		_ = auth.DeleteSession(h.store.DB(), c.Value)
	}
	auth.ClearSessionCookie(w, h.secure())
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) Account(w http.ResponseWriter, r *http.Request) {
	pd := h.pageData(r)
	pd.Title = "Account"
	h.render(w, r, "account", pd)
}

func (h *Handler) renderLoginError(w http.ResponseWriter, r *http.Request, msg string) {
	pd := h.pageData(r)
	pd.Title = "Sign in"
	pd.Message = &render.FlashMessage{Type: "error", Text: msg}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = render.Page(w, "login", pd)
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, pd *render.PageData) {
	if err := render.Page(w, name, pd); err != nil {
		h.internalError(w, r, err)
	}
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("internal error",
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", middleware.RequestIDFrom(r.Context()),
		"err", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
