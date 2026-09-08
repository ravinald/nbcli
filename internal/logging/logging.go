// Package logging configures structured logging for nbcli.
//
// Convention: --verbose flips the slog level from INFO to DEBUG and turns on
// source-location attributes. The default handler is text, written to stderr.
// Commands and the Netbox client emit slog.Debug for request lifecycle and
// slog.Info for state changes; errors flow back through return values, not
// through the logger.
package logging

import (
	"io"
	"log/slog"
)

// Setup installs a structured logger as slog's default and returns it.
// Pass io.Discard as w to silence logging entirely (useful for tests).
func Setup(verbose bool, w io.Writer) *slog.Logger {
	logger := slog.New(Handler(verbose, w))
	slog.SetDefault(logger)
	return logger
}

// Handler builds the slog.Handler with verbosity-appropriate options.
// Pulled out so callers can stack their own middleware (e.g. samber/slog-multi
// for fan-out) without re-implementing the level/source decisions.
func Handler(verbose bool, w io.Writer) slog.Handler {
	return slog.NewTextHandler(w, &slog.HandlerOptions{
		Level:     Level(verbose),
		AddSource: verbose,
	})
}

// Level returns slog.LevelDebug when verbose is true, otherwise slog.LevelInfo.
func Level(verbose bool) slog.Level {
	if verbose {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// Discard returns a no-op slog logger. Use in tests that exercise code paths
// emitting log lines but don't want the noise.
func Discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
}
