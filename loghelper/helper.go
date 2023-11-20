package loghelper

import (
	"context"
	"log/slog"
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
