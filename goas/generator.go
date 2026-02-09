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
	if w == nil {
		return errors.New("nil writer")
	}

	spec, err := buildSpec(ctx, opt)
	if err != nil {
		return err
	}

	b, err := openapi.Marshal(spec, openapi.MarshalOptions{Indent: "  "})
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func Generate(ctx context.Context, opt Options) ([]byte, error) {
	spec, err := buildSpec(ctx, opt)
	if err != nil {
		return nil, err
	}
	return openapi.Marshal(spec, openapi.MarshalOptions{Indent: "  "})
}
