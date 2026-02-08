package annotate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustCommentGroup(t *testing.T, src string) *ast.CommentGroup {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	require.NoError(t, err)

	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if gen.Doc != nil {
			return gen.Doc
		}
	}

	t.Fatalf("no comment group found in source")
	return nil
}

func Test_parseApiSchemaName(t *testing.T) {
	t.Run("comment without @ApiSchemaName", func(t *testing.T) {
		cg := mustCommentGroup(t, `
package p

// Some comment
type X struct{}
`)

		alias, ok, err := ParseApiSchemaName(cg)

		require.NoError(t, err)
		require.False(t, ok)
		require.Equal(t, "", alias)
	})

	t.Run("@ApiSchemaName without alias", func(t *testing.T) {
		cg := mustCommentGroup(t, `
package p

// @ApiSchemaName
type X struct{}
`)

		alias, ok, err := ParseApiSchemaName(cg)

		require.Error(t, err)
		require.False(t, ok)
		require.Equal(t, "", alias)
	})

	t.Run("@ApiSchemaName with alias", func(t *testing.T) {
		cg := mustCommentGroup(t, `
package p

// @ApiSchemaName Foo
type X struct{}
`)

		alias, ok, err := ParseApiSchemaName(cg)

		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, "Foo", alias)
	})
}
