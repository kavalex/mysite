package logging

import (
	"log/slog"
	"os"
)

// level is a global level that can be changed live (SIGHUP).
var level = new(slog.LevelVar)

func Init(env string) {
	level.Set(levelFor(env))
	var handler slog.Handler
	if env == "prod" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler))
}

// SetLevel applies a new level from SIGHUP-reload.
func SetLevel(env string) {
	level.Set(levelFor(env))
}

func levelFor(env string) slog.Level {
	if env == "prod" {
		return slog.LevelInfo
	}
	return slog.LevelDebug
}
