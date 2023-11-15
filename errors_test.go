package errors

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testErr struct {
	msg string
}

func (e *testErr) Error() string {
	return e.msg
}

func TestNewError(t *testing.T) {
	assert.Error(t, New("new error"))
}

func TestErrorAttr(t *testing.T) {

	err := New("error",
		slog.Int("a", 1),
		slog.Int("b", 2),
	)

	if assert.Error(t, err) {
		if lv, ok := err.(slog.LogValuer); assert.True(t, ok) {
			if v := lv.LogValue(); assert.Equal(t, slog.KindGroup, v.Kind()) {
				var keys []string
				for _, a := range lv.LogValue().Group() {
					keys = append(keys, a.Key)
				}
				slices.Sort(keys)
				assert.Equal(t, []string{"a", "b", "error"}, keys)
			}
		}
	}
}

func TestErrorIdentity(t *testing.T) {
	var (
		err1 = New("new error")
		err2 = New("new error",
			slog.String("a", "a"),
		)
	)

	assert.True(t, Is(err1, err1))
	assert.True(t, Is(err2, err2))
	assert.False(t, Is(err1, err2))
}

func TestWrappedError(t *testing.T) {

	var (
		errMsg        = "error message"
		origErr       = &testErr{msg: errMsg}
		compatibleErr *testErr
		osErr         *os.PathError
	)

	err := Wrap(origErr, "my error")
	assert.Error(t, err)
	assert.True(t, Is(err, origErr))
	assert.Equal(t, Unwrap(err), origErr)

	assert.True(t, As(err, &compatibleErr))
	assert.Equal(t, compatibleErr.msg, errMsg)
	assert.False(t, As(err, &osErr))
}

func TestWrappedErrorWithLogAttr(t *testing.T) {

	err := New("new error", slog.Int("a", 1))
	err = Wrap(err, "second error", slog.Int("b", 2))
	err = Wrap(err, "third error", slog.Int("c", 3))

	assert.Error(t, err)

	if lv, ok := err.(slog.LogValuer); assert.True(t, ok) {
		if v := lv.LogValue(); assert.Equal(t, slog.KindGroup, v.Kind()) {
			var keys []string
			for _, a := range lv.LogValue().Group() {
				keys = append(keys, a.Key)
			}
			slices.Sort(keys)
			assert.Equal(t, []string{"a", "b", "c", "error"}, keys)
		}
	}
}

type testData struct {
	err         error
	want        error
	description string
}

func TestErrorComparison(t *testing.T) {
	err := New("error")
	assert.Error(t, err)

	assert.Equal(t, err, New("error"))
	assert.NotEqual(t, err, New("another error"))

	for _, td := range []testData{
		{
			err:         nil,
			want:        nil,
			description: "nil error is nil",
		},
		{
			err:         (error)(nil),
			want:        nil,
			description: "explicit nil error is nil",
		},
		{
			err:         (*testErr)(nil),
			want:        (*testErr)(nil),
			description: "typed nil is nil",
		},
		{
			err:         New("error"),
			want:        New("error"),
			description: "same error value",
		},
		{
			err:         err,
			want:        err,
			description: "error returned from New() is unaffected",
		},
		{
			err:         io.EOF,
			want:        io.EOF,
			description: "unwrapped error is unaffected",
		},
		{
			err:         Wrap(io.EOF, "wrapped"),
			want:        io.EOF,
			description: "wrapped EOF error returns what is wrapped",
		},
		{
			err:         Wrap(err, "wrapped"),
			want:        err,
			description: "wrapped error error returns what is wrapped",
		},
		{
			description: "wrapped with log attributes",
			err:         Wrap(err, "wrapped", slog.String("a", "b")),
			want:        err,
		},
		{
			description: "wrapped multiple times",
			err: Wrap(Wrap(fmt.Errorf("stderrors wrap: %w", err), "errors wrap"),
				"errors wrap", slog.String("a", "b")),
			want: err,
		},
	} {
		t.Run(td.description, func(t *testing.T) {
			assert.EqualValues(t, Unwrap(td.err), Unwrap(td.want))
			assert.True(t, Is(td.want, td.want))
		})
	}
}
