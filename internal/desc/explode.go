package desc

import (
	"context"

	"github.com/delley/goas/internal/openapi"
)

func ExplodeRefs(fileRefPath string, spec *openapi.OpenAPIObject) error {
	return ExplodeRefsContext(context.Background(), fileRefPath, spec)
}

func ExplodeRefsContext(ctx context.Context, fileRefPath string, spec *openapi.OpenAPIObject) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if spec.Info.Description != nil {
		desc, err := FetchRefContext(ctx, fileRefPath, spec.Info.Description.Value)
		if err != nil {
			return err
		}
		spec.Info.Description.Value = desc
	}
	for i, tag := range spec.Tags {
		if err := ctx.Err(); err != nil {
			return err
		}
		if tag.Description == nil {
			continue
		}
		desc, err := FetchRefContext(ctx, fileRefPath, tag.Description.Value)
		if err != nil {
			return err
		}
		spec.Tags[i].Description.Value = desc
	}

	return nil
}
