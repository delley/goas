package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/delley/goas/goas"
	"github.com/urfave/cli"
)

var version = "v1.0.0"

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

func action(c *cli.Context) error {
	opts := goas.Options{
		ModulePath:   c.GlobalString("module-path"),
		MainFilePath: c.GlobalString("main-file-path"),
		HandlerPath:  c.GlobalString("handler-path"),
		FileRefPath:  c.GlobalString("file-ref-path"),
		OutputPath:   c.GlobalString("output"),
		Debug:        c.GlobalBool("debug"),
		OmitPackages: c.GlobalBool("omit-packages"),
		ShowHidden:   c.GlobalBool("show-hidden"),
	}

	var w io.Writer = os.Stdout
	if opts.OutputPath != "" && opts.OutputPath != "-" {
		f, err := os.Create(opts.OutputPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	gen := goas.New()
	return gen.GenerateTo(context.Background(), opts, w)
}

func main() {
	app := cli.NewApp()
	app.Name = "goas"
	app.Usage = ""
	app.UsageText = "goas [options]"
	app.Version = version
	app.Copyright = "(c) 2026 delley.fx@gmail.com"
	app.HideHelp = true
	app.OnUsageError = func(c *cli.Context, err error, isSubcommand bool) error {
		cli.ShowAppHelp(c)
		return nil
	}
	app.Flags = flags
	app.Action = action

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal("Error: ", err)
	}
}
