package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sort"
	"strings"

	"github.com/vovanec/errors/internal"
)

const (
	errKey       = "error"
	msgKey       = "msg"
	errOriginKey = "origin"
)

type sError struct {
	err    error
	origin ErrorOrigin
	attrs  map[string]slog.Attr
}

func (e *sError) LogValue() slog.Value {

	errGroup := slog.Group(errKey, slog.String(msgKey, e.err.Error()))
	if !e.origin.Empty() {
		errGroup = slog.Group(errKey,
			slog.String(msgKey, e.err.Error()),
			slog.String(errOriginKey, e.origin.String()),
		)
	}
	attrs := append(internal.MapValues(e.attrs), errGroup)

	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Key < attrs[j].Key
	})

	return slog.GroupValue(attrs...)
}

func (e *sError) Origin() ErrorOrigin {
	return e.origin
}

func (e *sError) StructuredError() string {
	if len(e.attrs) < 1 {
		return e.err.Error()
	}

	attrs := internal.MapValues(e.attrs)
	sort.Slice(attrs, func(i, j int) bool {
		return attrs[i].Key < attrs[j].Key
	})

	var formattedParts []string
	for _, a := range attrs {
		if a.Key == errKey || a.Key == msgKey {
			continue
		}
		formattedParts = append(formattedParts, a.String())
	}

	if len(formattedParts) < 1 {
		return e.err.Error()
	}

	return fmt.Sprintf("%s: %s", e.err.Error(), strings.Join(formattedParts, " "))
}

func (e *sError) Error() string {
	return e.err.Error()
}

func (e *sError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') || s.Flag('#') {

			_, _ = fmt.Fprint(s, e.StructuredError())
			return
		}

		fallthrough
	case 's':
		_, _ = fmt.Fprint(s, e.Error())
	}
}

// Unwrap returns the result of calling the Unwrap method on err, if error's
// type contains an Unwrap method returning error (otherwise nil).
func (e *sError) Unwrap() error {
	return e.err
}

// New returns an error that formats as the given text with optional log args.
func New(message string, args ...any) error {

	am := make(map[string]slog.Attr)
	internal.ParseLogArgs(
		args,
		func(a slog.Attr) {
			am[a.Key] = a
		},
	)

	if len(am) < 1 {
		return errors.New(message)
	}

	return &sError{
		err:    errors.New(message),
		attrs:  am,
		origin: getOrigin(2),
	}
}

// Wrap wraps the original error and new returned error will implement an Unwrap interface.
// This also will add log args to the error if there are any.
func Wrap(err error, message string, args ...any) error {

	if err == nil {
		return nil
	}

	am := make(map[string]slog.Attr)
	internal.ParseLogArgs(
		append([]any{err}, args...),
		func(a slog.Attr) {
			if a.Key != errKey {
				am[a.Key] = a
			}
		},
	)

	if len(am) < 1 {
		return fmt.Errorf("%s: %w", message, err)
	}

	var (
		sErr   *sError
		origin ErrorOrigin
	)
	if As(err, &sErr) {
		origin = sErr.origin
	} else {
		origin = getOrigin(2)
	}

	return &sError{
		err:    fmt.Errorf("%s: %w", message, err),
		attrs:  am,
		origin: origin,
	}
}

// Unwrap returns the result of recursive calling the Unwrap method on err, if error's
// type contains an Unwrap method returning error (the original error will be returned otherwise).
func Unwrap(err error) error {
	for err != nil {
		if u, ok := err.(interface{ Unwrap() error }); !ok {
			break
		} else {
			err = u.Unwrap()
		}
	}
	return err
}

// Is reports whether any error in error's chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in error's chain that matches target, and if so, sets
// target to that error value and returns true. Otherwise, it returns false.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

func getOrigin(n int) ErrorOrigin {
	if _, file, line, ok := runtime.Caller(n); ok {
		return ErrorOrigin{
			Line: line,
			File: file,
		}
	}
	return ErrorOrigin{}
}

type ErrorOrigin struct {
	Line int
	File string
}

func (o ErrorOrigin) String() string {
	return fmt.Sprintf("%s:%d", o.File, o.Line)
}

func (o ErrorOrigin) Empty() bool {
	return o.File == ""
}
