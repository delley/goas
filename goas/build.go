package goas

import (
	"context"

	"github.com/delley/goas/internal/desc"
	generatepkg "github.com/delley/goas/internal/generate"
	"github.com/delley/goas/internal/openapi"
	internalSchema "github.com/delley/goas/internal/schema"
)

// buildSpec is the generation orchestrator. It coordinates the internal
// parser, schema validation, and reference expansion without exposing those
// implementation details through the public Generator API.
func buildSpec(ctx context.Context, opt Options) (*openapi.OpenAPIObject, error) {
	pipeline := generatepkg.NewPipeline(ctx, generatepkg.NewDebugLogger(opt.Debug))
	input := generatepkg.Input{
		ModulePath:   opt.ModulePath,
		MainFilePath: opt.MainFilePath,
		HandlerPath:  opt.HandlerPath,
		FileRefPath:  opt.FileRefPath,
		OmitPackages: opt.OmitPackages,
		ShowHidden:   opt.ShowHidden,
		Debug:        opt.Debug,
	}

	spec, err := pipeline.Run(input,
		func(ctx context.Context, in generatepkg.Input) (*generatepkg.Document, error) {
			p, err := newParserContext(ctx,
				in.ModulePath,
				in.MainFilePath,
				in.HandlerPath,
				in.FileRefPath,
				in.Debug,
				in.OmitPackages,
				in.ShowHidden,
			)
			if err != nil {
				return nil, err
			}
			if err := p.parse(); err != nil {
				return nil, err
			}
			return &generatepkg.Document{Spec: &p.OpenAPI}, nil
		},
		generatepkg.Phase{
			Name: "validate-schema-names",
			Run: func(ctx context.Context, doc *generatepkg.Document) error {
				p, err := newParserContext(ctx,
					input.ModulePath,
					input.MainFilePath,
					input.HandlerPath,
					input.FileRefPath,
					input.Debug,
					input.OmitPackages,
					input.ShowHidden,
				)
				if err != nil {
					return err
				}
				p.OpenAPI = *doc.Spec
				p.schemaRegistry = internalSchema.NewRegistry(input.OmitPackages)
				p.KnownIDSchema = p.schemaRegistry.KnownIDSchema
				p.ApiSchemaNames = p.schemaRegistry.ApiSchemaNames
				return p.validateSchemaNames()
			},
		},
		generatepkg.Phase{
			Name: "expand-description-refs",
			Run: func(ctx context.Context, doc *generatepkg.Document) error {
				return desc.ExplodeRefsContext(ctx, input.FileRefPath, doc.Spec)
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return spec, nil
}
