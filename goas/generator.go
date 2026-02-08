package goas

import (
	"context"
	"io"

	"github.com/delley/goas"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateTo(ctx context.Context, opt Options, w io.Writer) error {
	p, err := goas.NewParser(opt.ModulePath, opt.MainFilePath, opt.HandlerPath, opt.FileRefPath, opt.Debug, opt.OmitPackages, opt.ShowHidden)
	if err != nil {
		return err
	}

	return p.CreateOASFile(opt.OutputPath)
}
