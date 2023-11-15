package loghelper

import (
	"context"
	"log/slog"
)

func Error(err error) slog.Attr {
	if lv, ok := err.(slog.LogValuer); ok {
		return slog.Any("", lv.LogValue())
	}
	return slog.Attr{}
}

func Context(ctx context.Context) slog.Attr {
	var ret []any
	for _, a := range logAttrsFromContext(ctx) {
		ret = append(ret, a)
	}
	return slog.Group("", ret...)
}

func WithAttrs(ctx context.Context, attr ...any) context.Context {

	var (
		attrMap = logAttrsFromContext(ctx)
		record  slog.Record
	)

	record.Add(attr...)
	record.Attrs(func(a slog.Attr) bool {
		attrMap[a.Key] = a
		return true
	})

	return context.WithValue(
		ctx,
		logAttrCtxKey,
		attrMap,
	)
}

type (
	logAttrCtxKeyType struct{}
)

var logAttrCtxKey logAttrCtxKeyType

func logAttrsFromContext(ctx context.Context) map[string]slog.Attr {
	if attr, ok := ctx.Value(logAttrCtxKey).(map[string]slog.Attr); ok {
		return attr
	}
	return make(map[string]slog.Attr)
}
