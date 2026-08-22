package main

import (
	"context"
	"io"
	"log"
	"os"

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

func outputWriter(opts goas.Options) (io.Writer, func(), error) {
	if opts.OutputPath == "" || opts.OutputPath == "-" {
		return os.Stdout, func() {}, nil
	}

	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return nil, nil, err
	}

	return f, func() {
		_ = f.Close()
	}, nil
}

func action(c *cli.Context) error {
	opts := optionsFromContext(c)
	w, cleanup, err := outputWriter(opts)
	if err != nil {
		return err
	}
	defer cleanup()

	gen := goas.New()
	return gen.GenerateTo(context.Background(), opts, w)
}

func before(c *cli.Context) error {
	if c.NArg() == 0 && len(os.Args) == 1 {
		_ = cli.ShowAppHelp(c)
		return cli.NewExitError("", 0)
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
		_ = cli.ShowAppHelp(c)
		return nil
	}
	return app
}

func main() {
	if err := newApp().Run(os.Args); err != nil {
		log.Fatal("Error: ", err)
	}
}
