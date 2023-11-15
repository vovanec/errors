package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/vovanec/errors"
	"github.com/vovanec/errors/loghelper"
)

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

func doSomethingElse(ctx context.Context) error {

	slog.Info("logging in doSomethingElse",
		loghelper.Context(ctx))

	return errors.New("error in doSomethingElse",
		slog.String("a", "a"),
	)
}

func doSomething(ctx context.Context) error {
	if err := doSomethingElse(ctx); err != nil {
		return errors.Wrap(err, "error in doSomething",
			slog.String("b", "b"),
		)
	}
	return nil
}

func main() {

	slog.SetDefault(
		slog.New(
			slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		),
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

	ctx := loghelper.WithAttrs(context.Background(),
		slog.Any("application", app),
	)

	if err := doSomething(ctx); err != nil {
		slog.ErrorContext(ctx, "error",
			loghelper.Context(ctx),
			loghelper.Error(err),
		)
	}
}
