package goas

import (
	"context"

	"github.com/delley/goas/internal/desc"
	"github.com/delley/goas/internal/openapi"
)

// buildSpec is the generation orchestrator. It coordinates the internal
// parser, schema validation, and reference expansion without exposing those
// implementation details through the public Generator API.
func buildSpec(ctx context.Context, opt Options) (*openapi.OpenAPIObject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, err := newParserContext(ctx,
		opt.ModulePath,
		opt.MainFilePath,
		opt.HandlerPath,
		opt.FileRefPath,
		opt.Debug,
		opt.OmitPackages,
		opt.ShowHidden,
	)
	if err != nil {
		return nil, err
	}

	if err := p.parse(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := p.validateSchemaNames(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := desc.ExplodeRefsContext(ctx, p.FileRefPath, &p.OpenAPI); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &p.OpenAPI, nil
}
