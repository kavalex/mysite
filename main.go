package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"github.com/kavalex/mysite/internal/config"
	"github.com/kavalex/mysite/internal/logging"
	"github.com/kavalex/mysite/internal/server"
	"github.com/kavalex/mysite/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// Dev convenience: .env is read but does not override already-set env.
	_ = config.LoadEnvFile(".env")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logging.Init(cfg.Env)
	slog.Info("starting", "env", cfg.Env, "addr", cfg.Addr)

	s, err := store.Open(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	if err := s.Migrate(); err != nil {
		return err
	}

	srv, rt, err := server.New(cfg, s)
	if err != nil {
		return err
	}
	defer rt.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// SIGHUP — graceful config reload.
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	go handleReload(hup, cfg, rt)

	errCh := make(chan error, 1)
	go func() {
		var err error
		if cfg.TLS != nil {
			err = srv.ListenAndServeTLS(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// handleReload reacts to SIGHUP: re-reads .env, applies reloadable fields
// (ENV, LOGIN_*), logs the rest as "requires restart".
func handleReload(hup <-chan os.Signal, cfg *config.Config, rt *server.Runtime) {
	for range hup {
		slog.Info("SIGHUP: reloading config")

		if err := config.ForceReloadEnvFile(".env"); err != nil {
			slog.Error("SIGHUP: env reload failed", "err", err)
			continue
		}
		next, err := config.Load()
		if err != nil {
			slog.Error("SIGHUP: config invalid, keeping old", "err", err)
			continue
		}

		logging.SetLevel(next.Env)
		rt.LoginRL.SetLimit(rate.Limit(float64(next.LoginRatePerMin) / 60.0))
		rt.LoginRL.SetBurst(next.LoginBurst)

		for _, field := range cfg.Diff(next) {
			switch field {
			case "ENV", "LOGIN_RATE_PER_MIN", "LOGIN_BURST":
				slog.Info("SIGHUP: applied", "field", field)
			default:
				slog.Warn("SIGHUP: change requires restart", "field", field)
			}
		}

		cfg.Env = next.Env
		cfg.LoginRatePerMin = next.LoginRatePerMin
		cfg.LoginBurst = next.LoginBurst
	}
}
