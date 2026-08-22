package schema

import (
	"go/ast"
	"testing"

	"github.com/delley/goas/internal/openapi"
	"github.com/stretchr/testify/require"
)

func TestRegistrySchemaNamesAndConflicts(t *testing.T) {
	registry := NewRegistry(false)
	registry.ApiSchemaNames["example.com/one"] = map[string]string{"Thing": "Shared"}
	registry.ApiSchemaNames["example.com/two"] = map[string]string{"Other": "Shared"}

	require.Error(t, registry.ValidateSchemaNames())
	generated := NewRegistry(false)
	require.Equal(t, "one.Thing", generated.SchemaObjectID("example.com/one", "Thing"))
}

func TestRegistryOmitPackagesAndDeduplicatesSchemas(t *testing.T) {
	registry := NewRegistry(true)
	first := &openapi.SchemaObject{ID: "Thing"}
	second := &openapi.SchemaObject{ID: "Thing"}

	require.NoError(t, registry.RegisterSchemaObject("example.com/one", "pkg.Thing", first))
	require.NoError(t, registry.RegisterSchemaObject("example.com/one", "Thing", second))
	require.Same(t, first, registry.KnownIDSchema["Thing"])
	require.Equal(t, "Thing", registry.SchemaObjectID("example.com/one", "Thing"))
}

func TestRegistryParsesSchemaAnnotation(t *testing.T) {
	registry := NewRegistry(false)
	comments := &ast.CommentGroup{List: []*ast.Comment{{Text: "// @ApiSchemaName PublicThing"}}}

	require.NoError(t, registry.ParseTypeAnnotations("example.com/types", "Thing", comments))
	require.Equal(t, "PublicThing", registry.SchemaObjectID("example.com/types", "Thing"))
}
