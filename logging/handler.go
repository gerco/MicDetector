//go:build !darwin

package logging

import (
	"log/slog"
	"os"
)

// NewHandler creates a Handler that logs to stdout.
// On non-Darwin platforms, this uses the standard library's text handler.
func NewHandler(subsystem, category string, level slog.Level) slog.Handler {
	return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
}
