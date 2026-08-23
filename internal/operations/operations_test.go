package operations

import (
	"context"
	"testing"

	"github.com/delley/goas/internal/openapi"
	"github.com/stretchr/testify/require"
)

func TestParseRouteCommentRejectsInvalidInput(t *testing.T) {
	_, err := ParseRouteComment("@route /users")
	require.Error(t, err)
}

func TestApplyRouteRejectsDuplicateMethod(t *testing.T) {
	doc := &openapi.OpenAPIObject{
		Paths: openapi.PathsObject{
			"/users": {
				Get: &openapi.OperationObject{Responses: openapi.ResponsesObject{}},
			},
		},
	}

	op := &openapi.OperationObject{Responses: openapi.ResponsesObject{}}
	err := ApplyRoute(doc, op, "/users", "GET")
	require.Error(t, err)
}

func TestParseRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := Parser{FileRefPath: "."}
	_, err := p.Parse(ctx, Input{Comments: []string{"@description $ref:file://../../example/example.md"}})
	require.ErrorIs(t, err, context.Canceled)
}
