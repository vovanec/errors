package errors

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/vovanec/errors/internal"
)

const errKey = "error"

type cpError struct {
	err   error
	attrs map[string]slog.Attr
}

func (e *cpError) LogValue() slog.Value {
	return slog.GroupValue(
		append(
			internal.ToSlice(e.attrs),
			slog.String(errKey, e.err.Error()),
		)...,
	)
}

func (e *cpError) Error() string {
	return e.err.Error()
}

// Unwrap returns the result of calling the Unwrap method on err, if error's
// type contains an Unwrap method returning error (otherwise nil).
func (e *cpError) Unwrap() error {
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

	return &cpError{
		err:   errors.New(message),
		attrs: am,
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

	return &cpError{
		err:   fmt.Errorf("%s: %w", message, err),
		attrs: am,
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
