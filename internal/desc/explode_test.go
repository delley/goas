package desc

import (
	"testing"

	"github.com/delley/goas/internal/openapi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func Test_explodeRefs(t *testing.T) {
	t.Run("Info.Description unchanged when not a ref", func(t *testing.T) {

		o := openapi.OpenAPIObject{}
		o.Info.Description = &openapi.ReffableString{Value: "Foo"}

		err := ExplodeRefs("../../example/main.go", &o)
		require.NoError(t, err)

		require.Equal(t, "Foo", o.Info.Description.Value)
	})

	t.Run("Info.Description inlined when a ref", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "https://example.com",
			httpmock.NewStringResponder(200, "The quick brown fox jumped over the lazy dog"))
		o := openapi.OpenAPIObject{}
		o.Info.Description = &openapi.ReffableString{Value: "$ref:https://example.com"}

		err := ExplodeRefs("../../example/main.go", &o)
		require.NoError(t, err)

		require.Equal(t, "The quick brown fox jumped over the lazy dog", o.Info.Description.Value)
	})

	t.Run("Tags[].Description unchanged when not a ref", func(t *testing.T) {
		o := openapi.OpenAPIObject{}
		o.Tags = []openapi.TagDefinition{{Name: "Foo", Description: &openapi.ReffableString{Value: "Foobar"}}}

		err := ExplodeRefs("../../example/main.go", &o)
		require.NoError(t, err)

		require.Equal(t, "Foobar", o.Tags[0].Description.Value)
	})

	t.Run("Tags[].Description inlined when a ref", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "https://example.com",
			httpmock.NewStringResponder(200, "The quick brown fox jumped over the lazy dog"))
		o := openapi.OpenAPIObject{}
		o.Tags = []openapi.TagDefinition{{Name: "Foo", Description: &openapi.ReffableString{Value: "$ref:https://example.com"}}}

		err := ExplodeRefs("../../example/main.go", &o)
		require.NoError(t, err)

		require.Equal(t, "The quick brown fox jumped over the lazy dog", o.Tags[0].Description.Value)
	})

	t.Run("Mixed of tag refs and non-refs", func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "https://example.com",
			httpmock.NewStringResponder(200, "The quick brown fox jumped over the lazy dog"))
		o := openapi.OpenAPIObject{}
		o.Tags = []openapi.TagDefinition{{Name: "Foo", Description: &openapi.ReffableString{Value: "$ref:https://example.com"}}, {Name: "Bar", Description: &openapi.ReffableString{Value: "Baz"}}}

		err := ExplodeRefs("../../example/main.go", &o)
		require.NoError(t, err)

		require.Equal(t, "The quick brown fox jumped over the lazy dog", o.Tags[0].Description.Value)
		require.Equal(t, "Baz", o.Tags[1].Description.Value)
	})
}
