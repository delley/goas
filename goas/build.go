package goas

import (
	"context"

	"github.com/delley/goas/internal/desc"
	"github.com/delley/goas/internal/openapi"
)

func buildSpec(ctx context.Context, opt Options) (*openapi.OpenAPIObject, error) {
	p, err := newParser(
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

	if err := p.validateSchemaNames(); err != nil {
		return nil, err
	}

	if err := desc.ExplodeRefs(p.FileRefPath, &p.OpenAPI); err != nil {
		return nil, err
	}

	return &p.OpenAPI, nil
}
