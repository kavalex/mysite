package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		defer func() {
			if err := recover(); err != nil {
				slog.ErrorContext(ctx, "panic recovered",
					"err", err,
					"stack", string(debug.Stack()),
					"path", r.URL.Path,
					"request_id", RequestIDFrom(ctx))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
