package errors

import (
	"errors"
	"fmt"
	"log/slog"
)

const errKey = "error"

type cpError struct {
	err  error
	attr map[string]slog.Attr
}

func (e *cpError) LogValue() slog.Value {
	return slog.GroupValue(
		append(
			toSlice(e.attr),
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

// New returns an error that formats as the given text with optional log attributes.
func New(message string, attr ...any) error {

	var record slog.Record
	record.Add(attr...)

	m := make(map[string]slog.Attr)
	record.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a
		return true
	})

	if len(m) < 1 {
		return errors.New(message)
	}

	return &cpError{
		err:  errors.New(message),
		attr: m,
	}
}

// Wrap wraps the original error and new returned error will implement an Unwrap interface.
// This also will add log attributes to the error if there are any.
func Wrap(err error, message string, attr ...any) error {

	if err == nil {
		return nil
	}

	newAttr := make(map[string]slog.Attr)
	if lv, ok := err.(slog.LogValuer); ok {
		for _, a := range lv.LogValue().Group() {
			if a.Key != errKey {
				newAttr[a.Key] = a
			}
		}
	}

	var record slog.Record
	record.Add(attr...)
	record.Attrs(func(a slog.Attr) bool {
		newAttr[a.Key] = a
		return true
	})

	if len(newAttr) < 1 {
		return fmt.Errorf("%s: %w", message, err)
	}

	return &cpError{
		err:  fmt.Errorf("%s: %w", message, err),
		attr: newAttr,
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

func toSlice[K comparable, V any](m map[K]V) []V {
	var ret []V
	for _, a := range m {
		ret = append(ret, a)
	}
	return ret
}
