package errors

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vovanec/errors/loghelper"
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

type AppVersion struct {
	Major int
	Minor int
	Patch int
}

func (v AppVersion) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("major", v.Major),
		slog.Int("minor", v.Minor),
		slog.Int("patch", v.Patch),
	)
}

type Application struct {
	Name    string
	Version AppVersion
	Build   string
}

func (a Application) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("name", a.Name),
		slog.Any("version", a.Version),
		slog.Group("build",
			slog.String("hash", a.Build),
		),
	)
}

type InlineArgs struct {
	Arg1 string
	Arg2 string
	Arg3 string
}

func (a InlineArgs) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("arg1", a.Arg1),
		slog.String("arg2", a.Arg2),
		slog.String("arg3", a.Arg3),
	)
}

const expectedLog = `{"time":"","level":"INFO","msg":"application started","application":{"name":"vovan","version":{"major":1,"minor":7,"patch":2},"build":{"hash":"20b8c3f"}},"arg1":"ARG1","arg2":"ARG2","arg3":"ARG3","x":"x"}
{"time":"","level":"INFO","msg":"logging in doSomethingElse","application":{"name":"vovan","version":{"major":1,"minor":7,"patch":2},"build":{"hash":"20b8c3f"}},"arg1":"ARG1","arg2":"ARG2","arg3":"ARG3"}
{"time":"","level":"ERROR","msg":"error occurred","a":"a","application":{"name":"vovan","version":{"major":1,"minor":7,"patch":2},"build":{"hash":"20b8c3f"}},"arg1":"ARG1","arg2":"ARG2","arg3":"ARG3","b":"b","c":"c","error":"error in doSomething: error in doSomethingElse"}
`

func TestErrorLogging(t *testing.T) {

	var buf bytes.Buffer
	logger := slog.New(
		slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelInfo,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == "time" {
					a.Value = slog.StringValue("")
				}
				return a
			},
		}),
	)

	app := Application{
		Name:  "vovan",
		Build: "20b8c3f",
		Version: AppVersion{
			Major: 1,
			Minor: 7,
			Patch: 2,
		},
	}

	inlineArgs := InlineArgs{
		Arg1: "ARG1",
		Arg2: "ARG2",
		Arg3: "ARG3",
	}

	ctx := loghelper.Context(context.Background(), inlineArgs, "application", app)

	// loghelper.Attr can be used instead of slog attribute constructors
	// if we want to extract log attributes from context or errors.
	logger.Info("application started",
		loghelper.Attr(
			ctx,
			slog.String("x", "x"),
		),
	)

	err := doSomething(ctx, logger)
	assert.Error(t, err)
	logger.Error("error occurred",
		loghelper.Attr(ctx, err),
	)

	assert.Equal(t, expectedLog, buf.String())
}

func doSomethingElse(ctx context.Context, logger *slog.Logger) error {

	logger.Info("logging in doSomethingElse",
		loghelper.Attr(ctx))

	return New("error in doSomethingElse",
		slog.String("a", "a"),
	)
}

func doSomething(ctx context.Context, logger *slog.Logger) error {
	if err := doSomethingElse(ctx, logger); err != nil {
		return Wrap(err, "error in doSomething",
			loghelper.Attr(
				// usually one doesn't have to attach the context since caller
				// already has it, but it can be done.
				ctx,
				slog.String("b", "b"),
				slog.String("c", "c"),
			),
		)
	}
	return nil
}
