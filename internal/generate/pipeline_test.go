package generate

import (
	"context"
	"testing"

	"github.com/delley/goas/internal/openapi"
	"github.com/stretchr/testify/require"
)

func TestPipelineRunsStagesInOrder(t *testing.T) {
	ctx := context.Background()
	calls := make([]string, 0, 3)
	spec := &openapi.OpenAPIObject{OpenAPI: openapi.OpenAPIVersion}

	result, err := NewPipeline(ctx, nil).Run(Input{}, func(ctx context.Context, in Input) (*Document, error) {
		calls = append(calls, "build")
		return &Document{Spec: spec}, nil
	},
		Phase{Name: "validate", Run: func(ctx context.Context, doc *Document) error {
			calls = append(calls, "validate")
			return nil
		}},
		Phase{Name: "expand", Run: func(ctx context.Context, doc *Document) error {
			calls = append(calls, "expand")
			return nil
		}},
	)
	require.NoError(t, err)
	require.Same(t, spec, result)
	require.Equal(t, []string{"build", "validate", "expand"}, calls)
}

func TestPipelineCancellationStopsBeforeAndDuringStages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewPipeline(ctx, nil).Run(Input{}, func(ctx context.Context, in Input) (*Document, error) {
		return &Document{Spec: &openapi.OpenAPIObject{}}, nil
	}, Phase{Name: "validate", Run: func(ctx context.Context, doc *Document) error {
		return nil
	}})
	require.ErrorIs(t, err, context.Canceled)

	ctx, cancel = context.WithCancel(context.Background())
	cancelled := false
	_, err = NewPipeline(ctx, nil).Run(Input{}, func(ctx context.Context, in Input) (*Document, error) {
		return &Document{Spec: &openapi.OpenAPIObject{}}, nil
	}, Phase{Name: "validate", Run: func(ctx context.Context, doc *Document) error {
		cancelled = true
		cancel()
		return ctx.Err()
	}})
	require.True(t, cancelled)
	require.ErrorIs(t, err, context.Canceled)
}
