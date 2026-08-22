package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/delley/goas/goas"
	"github.com/urfave/cli"
)

const version = "v1.0.0"

var flags = []cli.Flag{
	cli.StringFlag{
		Name:  "module-path",
		Value: ".",
		Usage: "goas will search @comment under the module",
	},
	cli.StringFlag{
		Name:  "main-file-path",
		Value: "",
		Usage: "goas will start to search @comment from this main file",
	},
	cli.StringFlag{
		Name:  "handler-path",
		Value: "",
		Usage: "goas only search handleFunc comments under the path",
	},
	cli.StringFlag{
		Name:  "file-ref-path",
		Value: ".",
		Usage: "path to start looking for file refs",
	},
	cli.StringFlag{
		Name:  "output",
		Value: "oas.json",
		Usage: "output file",
	},
	cli.BoolFlag{
		Name:  "debug",
		Usage: "show debug message",
	},
	cli.BoolFlag{
		Name:  "omit-packages",
		Usage: "Omit packages from schema names. An error will be thrown if there is a conflict.",
	},
	cli.BoolFlag{
		Name:  "show-hidden",
		Usage: "Generate schema even for paths that are marked as hidden packages",
	},
}

func optionsFromContext(c *cli.Context) goas.Options {
	return goas.Options{
		ModulePath:   c.String("module-path"),
		MainFilePath: c.String("main-file-path"),
		HandlerPath:  c.String("handler-path"),
		FileRefPath:  c.String("file-ref-path"),
		OutputPath:   c.String("output"),
		Debug:        c.Bool("debug"),
		OmitPackages: c.Bool("omit-packages"),
		ShowHidden:   c.Bool("show-hidden"),
	}
}

func outputWriter(opts goas.Options) (io.Writer, string, func(), error) {
	if opts.OutputPath == "" || opts.OutputPath == "-" {
		return os.Stdout, "", func() {}, nil
	}

	dir := filepath.Dir(opts.OutputPath)
	if dir == "" {
		dir = "."
	}

	f, err := os.CreateTemp(dir, ".goas-*.tmp")
	if err != nil {
		return nil, "", nil, err
	}
	return f, f.Name(), func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}, nil
}

func action(c *cli.Context) error {
	if c.NArg() == 0 && len(os.Args) == 1 {
		return nil
	}

	opts := optionsFromContext(c)
	if opts.OutputPath == "" || opts.OutputPath == "-" {
		gen := goas.New()
		return gen.GenerateTo(context.Background(), opts, os.Stdout)
	}

	w, tmpPath, cleanup, err := outputWriter(opts)
	if err != nil {
		return err
	}
	defer cleanup()

	gen := goas.New()
	if err := gen.GenerateTo(context.Background(), opts, w); err != nil {
		return err
	}
	if err := w.(*os.File).Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, opts.OutputPath); err != nil {
		return err
	}
	cleanup() // remove residual temp when rename fails; otherwise tmp no longer exists
	return nil
}

func before(c *cli.Context) error {
	if c.NArg() == 0 && len(os.Args) == 1 {
		if err := cli.ShowAppHelp(c); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func newApp() *cli.App {
	app := cli.NewApp()
	app.Name = "goas"
	app.Usage = ""
	app.UsageText = "goas [options]"
	app.Version = version
	app.Copyright = "(c) 2026 delley.fx@gmail.com"
	app.HideHelp = false
	app.Flags = flags
	app.Before = before
	app.Action = action
	app.OnUsageError = func(c *cli.Context, err error, isSubcommand bool) error {
		if helpErr := cli.ShowAppHelp(c); helpErr != nil {
			return errors.Join(err, helpErr)
		}
		return err
	}
	return app
}

func run(args []string, stdout, stderr io.Writer) int {
	app := newApp()
	app.Writer = stdout
	app.ErrWriter = stderr

	err := app.Run(args)
	if err == nil {
		return 0
	}

	code := 1
	if exitErr, ok := err.(cli.ExitCoder); ok {
		code = exitErr.ExitCode()
	}
	if err.Error() != "" {
		_, _ = fmt.Fprintln(stderr, "Error:", err)
	}
	return code
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}
