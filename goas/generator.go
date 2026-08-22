package goas

import (
	"context"
	"errors"
	"io"

	"github.com/delley/goas/internal/openapi"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateTo(ctx context.Context, opt Options, w io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if w == nil {
		return errors.New("nil writer")
	}

	b, err := g.Generate(ctx, opt)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func (g *Generator) Generate(ctx context.Context, opt Options) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	spec, err := buildSpec(ctx, opt)
	if err != nil {
		return nil, err
	}
	return openapi.Marshal(spec, openapi.MarshalOptions{Indent: "  "})
}
