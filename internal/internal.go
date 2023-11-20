package internal

import (
	"context"
	"fmt"
	"log/slog"
)

type (
	logAttrCtxKeyType struct{}
	AttrFunc          func(a slog.Attr)
)

var logAttrCtxKey logAttrCtxKeyType

func ContextWithLogArgs(ctx context.Context, args ...any) context.Context {

	am := logAttrsFromContext(ctx)
	ParseLogArgs(args, func(a slog.Attr) {
		am[a.Key] = a
	})

	return context.WithValue(
		ctx,
		logAttrCtxKey,
		am,
	)
}

func ParseLogArgs(args []any, f AttrFunc) {

	am := make(map[string]slog.Attr)
	for len(args) > 0 {
		var attrs []slog.Attr
		attrs, args = argsToAttr(args)
		for _, a := range attrs {
			if isEmptyGroup(a.Value) {
				continue
			} else if a.Key == "" {
				if a.Value.Kind() == slog.KindGroup {
					for _, ga := range a.Value.Group() {
						am[ga.Key] = ga
					}
				} else {
					panic(fmt.Sprintf("invalid attr, non-group value without a key: %v", a.Value))
				}
			} else {
				am[a.Key] = a
			}
		}
	}

	for _, a := range am {
		f(a)
	}
}

func ToSlice[K comparable, V any](m map[K]V) []V {
	var ret []V
	for _, a := range m {
		ret = append(ret, a)
	}
	return ret
}

const badKey = "!BADKEY"

func argsToAttr(args []any) ([]slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return []slog.Attr{slog.String(badKey, x)}, nil
		}
		return []slog.Attr{slog.Any(x, args[1])}, args[2:]
	case context.Context:
		return ToSlice(logAttrsFromContext(x)), args[1:]
	case error:
		return logAttrsFromError(x), args[1:]
	case slog.Attr:
		return []slog.Attr{x}, args[1:]
	default:
		return []slog.Attr{slog.Any(badKey, x)}, args[1:]
	}
}

func isEmptyGroup(v slog.Value) bool {
	if v.Kind() != slog.KindGroup {
		return false
	}
	return len(v.Group()) == 0
}

func logAttrsFromError(err error) []slog.Attr {
	if lv, ok := err.(slog.LogValuer); ok {
		if v := lv.LogValue(); v.Kind() == slog.KindGroup {
			return v.Group()
		} else {
			panic(fmt.Sprintf("non-group value in error: %v", v))
		}
	}
	return nil
}

func logAttrsFromContext(ctx context.Context) map[string]slog.Attr {
	if attr, ok := ctx.Value(logAttrCtxKey).(map[string]slog.Attr); ok {
		return attr
	}
	return make(map[string]slog.Attr)
}
