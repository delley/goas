package goas

import (
	"context"
	"errors"
	"io"

	"github.com/delley/goas/internal/openapi"
)

// Generator is the public entry point for generating an OpenAPI document.
// It is intentionally stateless, so a value can be reused for multiple
// independent generations.
type Generator struct{}

// New creates a Generator with the default behavior.
func New() *Generator {
	return &Generator{}
}

// GenerateTo generates an OpenAPI document and writes its JSON representation
// to w. It does not close w.
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

// Generate generates an OpenAPI document as indented JSON.
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
