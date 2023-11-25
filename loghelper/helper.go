package loghelper

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sort"

	"github.com/vovanec/errors/internal"
)

// Attr parses log args and returns a either a single log attribute or unnamed group.
func Attr(args ...any) slog.Attr {

	var attrs []slog.Attr
	internal.ParseLogArgs(args, func(a slog.Attr) {
		attrs = append(attrs, a)
	})

	if len(attrs) < 1 {
		return slog.Attr{}
	} else if len(attrs) < 2 {
		return attrs[0]
	}

	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Key < attrs[j].Key
	})

	return slog.Attr{
		Key:   "",
		Value: slog.GroupValue(attrs...),
	}
}

// Context returns a copy of parent context with attached log args.
func Context(ctx context.Context, args ...any) context.Context {
	return internal.ContextWithLogArgs(ctx, args...)
}

type LogOption func(c *logConfig)

// WithLevel sets default logger log level.
func WithLevel(level slog.Level) LogOption {
	return func(c *logConfig) {
		c.level = level
	}
}

// WithOutput sets default logger log output.
func WithOutput(w io.Writer) LogOption {
	return func(c *logConfig) {
		c.output = w
	}
}

// InitLogging initializes default slog logger instance
// with info log level and stderr as a log output.
func InitLogging(opts ...LogOption) {
	conf := logConfig{
		level:  slog.LevelInfo,
		output: os.Stderr,
	}

	for _, opt := range opts {
		opt(&conf)
	}

	slog.SetDefault(
		slog.New(
			slog.NewJSONHandler(conf.output, &slog.HandlerOptions{
				Level: conf.level,
			}),
		),
	)
}

type logConfig struct {
	level  slog.Level
	output io.Writer
}
