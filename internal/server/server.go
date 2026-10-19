package server

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/kavalex/mysite/internal/auth"
	"github.com/kavalex/mysite/internal/config"
	"github.com/kavalex/mysite/internal/handlers"
	"github.com/kavalex/mysite/internal/middleware"
	"github.com/kavalex/mysite/internal/render"
	"github.com/kavalex/mysite/internal/store"
)

// Runtime holds resources that must be freed or reconfigured without
// recreating the server (SIGHUP).
type Runtime struct {
	LoginRL *middleware.RateLimiter
}

func (r *Runtime) Close() { r.LoginRL.Close() }

func New(cfg *config.Config, s *store.Store) (*http.Server, *Runtime, error) {
	h := handlers.New(cfg, s)
	mux := http.NewServeMux()

	// Static
	fileServer := http.FileServer(http.FS(render.StaticFS()))
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheControl(fileServer)))

	// Health checks
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.DB().PingContext(ctx); err != nil {
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	secure := cfg.TLS != nil || cfg.Env == "prod"

	// Rate limiter for /login
	loginRL, err := middleware.NewRateLimiter(
		rate.Limit(float64(cfg.LoginRatePerMin)/60.0),
		cfg.LoginBurst,
		cfg.TrustedProxies,
	)
	if err != nil {
		return nil, nil, err
	}
	rt := &Runtime{LoginRL: loginRL}

	// SEO
	mux.HandleFunc("GET /sitemap.xml", h.Sitemap)
	mux.HandleFunc("GET /robots.txt", h.Robots)

	// Public
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /about", h.About)

	// Auth
	mux.HandleFunc("GET /login", h.LoginForm)
	mux.Handle("POST /login", loginRL.Middleware(http.HandlerFunc(h.LoginSubmit)))
	mux.HandleFunc("POST /logout", h.Logout)

	// Protected
	requireLogin := auth.RequireLogin(s.DB(), secure)
	mux.Handle("GET /account", requireLogin(http.HandlerFunc(h.Account)))

	// CSRF: stdlib protection, Go 1.25+
	csrfProtection := http.NewCrossOriginProtection()
	// if needed: csrfProtection.AddTrustedOrigin("https://admin.example.com")

	// Middleware chain: outside → inside.
	// Execution: Recovery → RequestID → SecurityHeaders → MaxBodySize → Logging → CSRF → mux
	var handler http.Handler = mux
	handler = csrfProtection.Handler(handler) // CSRF wraps mux
	handler = middleware.Logging(handler)
	handler = middleware.MaxBodySize(1 << 20)(handler)
	handler = middleware.SecurityHeaders(cfg.Env)(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.Recovery(handler)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: cfg.HeaderTimeout,
	}
	return srv, rt, nil
}

func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		next.ServeHTTP(w, r)
	})
}
